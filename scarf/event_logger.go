package scarf

import (
    "encoding/json"
    "errors"
    "fmt"
    "log"
    "net/http"
    "net/url"
    "os"
    "runtime"
    "strings"
    "time"
)

const (
    defaultTimeout = 3 * time.Second
)

// sdkVersion is the SDK version embedded in the User-Agent.
// It can be overridden at build time via:
//   go build -ldflags "-X github.com/scarf-sh/scarf-go/scarf.sdkVersion=v1.2.3"
var sdkVersion = "0.1.0"

// ScarfEventLogger provides a simple API to send telemetry events to a Scarf endpoint.
type ScarfEventLogger struct {
    endpointURL    string
    defaultTimeout time.Duration
    disabled       bool
    verbose        bool
    httpClient     *http.Client
    logger         *log.Logger
}

// ErrDisabled is returned when analytics are disabled via environment settings.
var ErrDisabled = errors.New("scarf: analytics disabled by environment")

// NewScarfEventLogger creates a new logger with the required endpoint URL.
//
// Optionally pass a timeout to override the default (3 seconds).
//   logger := NewScarfEventLogger("https://your-endpoint", 5*time.Second)
func NewScarfEventLogger(endpointURL string, timeout ...time.Duration) *ScarfEventLogger {
    t := defaultTimeout
    if len(timeout) > 0 && timeout[0] > 0 {
        t = timeout[0]
    }

    verbose := envBool("SCARF_VERBOSE")
    disabled := envBool("DO_NOT_TRACK") || envBool("SCARF_NO_ANALYTICS")

    l := log.New(os.Stderr, "[scarf] ", log.LstdFlags)

    return &ScarfEventLogger{
        endpointURL:    endpointURL,
        defaultTimeout: t,
        disabled:       disabled,
        verbose:        verbose,
        httpClient: &http.Client{
            Timeout: t,
        },
        logger: l,
    }
}

// Enabled reports whether analytics are enabled.
func (s *ScarfEventLogger) Enabled() bool {
    return !s.disabled
}

// LogEvent sends an event using the logger's default timeout.
// Returns nil if the request completed successfully with a 2xx status code.
func (s *ScarfEventLogger) LogEvent(properties map[string]any) error {
    return s.logEventInternal(properties, s.defaultTimeout)
}

// LogEventWithTimeout sends an event using a custom timeout for this call.
// Returns nil if the request completed successfully with a 2xx status code.
func (s *ScarfEventLogger) LogEventWithTimeout(properties map[string]any, timeout time.Duration) error {
    if timeout <= 0 {
        timeout = s.defaultTimeout
    }
    return s.logEventInternal(properties, timeout)
}

func (s *ScarfEventLogger) logEventInternal(properties map[string]any, timeout time.Duration) error {
    if s.disabled {
        if s.verbose {
            s.logger.Println("analytics disabled via env; not sending event")
        }
        return ErrDisabled
    }

    if strings.TrimSpace(s.endpointURL) == "" {
        if s.verbose {
            s.logger.Println("no endpoint URL configured; aborting")
        }
        return errors.New("scarf: endpoint URL is required")
    }

    if properties == nil {
        properties = map[string]any{}
    }

    // Build URL with query parameters from properties
    u, err := url.Parse(s.endpointURL)
    if err != nil {
        if s.verbose {
            s.logger.Printf("invalid endpoint URL: %v\n", err)
        }
        return fmt.Errorf("scarf: invalid endpoint URL: %w", err)
    }

    q := u.Query()
    for k, v := range properties {
        str := stringifyParam(v)
        q.Set(k, str)
    }
    u.RawQuery = q.Encode()

    if s.verbose {
        s.logger.Printf("payload (query): %s\n", u.RawQuery)
    }

    req, err := http.NewRequest(http.MethodPost, u.String(), nil)
    if err != nil {
        if s.verbose {
            s.logger.Printf("failed to build request: %v\n", err)
        }
        return fmt.Errorf("scarf: build request: %w", err)
    }
    req.Header.Set("User-Agent", buildUserAgent())

    // Use per-call timeout without mutating the shared client.
    client := *s.httpClient
    client.Timeout = timeout

    if s.verbose {
        s.logger.Printf("sending event to %s (timeout=%s)\n", req.URL.String(), timeout)
    }

    resp, err := client.Do(req)
    if err != nil {
        if s.verbose {
            s.logger.Printf("request failed: %v\n", err)
        }
        return fmt.Errorf("scarf: request failed: %w", err)
    }
    defer func() {
        // Read and close the body defensively to allow connection reuse.
        // We don't need the response body content, so just ensure closure.
        _ = drainAndClose(resp)
    }()

    if resp.StatusCode >= 200 && resp.StatusCode < 300 {
        if s.verbose {
            s.logger.Printf("event logged successfully: %s\n", resp.Status)
        }
        return nil
    }

    if s.verbose {
        s.logger.Printf("non-success status: %s\n", resp.Status)
    }
    return fmt.Errorf("scarf: non-success status: %s", resp.Status)
}

func envBool(key string) bool {
    v := strings.TrimSpace(os.Getenv(key))
    if v == "" {
        return false
    }
    v = strings.ToLower(v)
    return v == "1" || v == "true" || v == "yes" || v == "on"
}

// drainAndClose ensures response bodies are closed; returns the first error encountered.
func drainAndClose(resp *http.Response) error {
    if resp == nil || resp.Body == nil {
        return nil
    }
    // We don't need to read the full body; just close it.
    err := resp.Body.Close()
    if err != nil {
        return err
    }
    return nil
}

// Validate configuration at runtime if needed.
func (s *ScarfEventLogger) validate() error {
    if strings.TrimSpace(s.endpointURL) == "" {
        return errors.New("scarf: endpoint URL is required")
    }
    return nil
}

func buildUserAgent() string {
    osName := runtime.GOOS
    if osName == "darwin" {
        osName = "macOS"
    }
    v := sdkVersion
    if strings.TrimSpace(v) == "" {
        v = "dev"
    }
    // Example: scarf-go/v1.2.3 (os=macOS; arch=arm64; go=1.22.3)
    goVer := strings.TrimPrefix(runtime.Version(), "go")
    return fmt.Sprintf("scarf-go/%s (os=%s; arch=%s; go=%s)", v, osName, runtime.GOARCH, goVer)
}

// stringifyParam converts a property value into a string suitable for URL query parameters.
// Simple types use fmt.Sprint; complex types are JSON-encoded.
func stringifyParam(v any) string {
    switch vv := v.(type) {
    case string:
        return vv
    case fmt.Stringer:
        return vv.String()
    default:
        // Try to JSON-encode complex types for stability.
        b, err := json.Marshal(v)
        if err == nil {
            // Use the JSON as-is for objects/arrays, but avoid quoting simple scalars twice.
            // If result is a quoted string, trim quotes for more natural query values.
            s := string(b)
            if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
                return s[1 : len(s)-1]
            }
            return s
        }
        return fmt.Sprint(v)
    }
}
