package scarf

import (
    "bytes"
    "encoding/json"
    "errors"
    "log"
    "net/http"
    "os"
    "strings"
    "time"
)

const (
    defaultTimeout = 3 * time.Second
    userAgent      = "scarf-go/0.1.0"
)

// ScarfEventLogger provides a simple API to send telemetry events to a Scarf endpoint.
type ScarfEventLogger struct {
    endpointURL    string
    defaultTimeout time.Duration
    disabled       bool
    verbose        bool
    httpClient     *http.Client
    logger         *log.Logger
}

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
// Returns true if the request completed successfully with a 2xx status code.
func (s *ScarfEventLogger) LogEvent(properties map[string]any) bool {
    return s.logEventInternal(properties, s.defaultTimeout)
}

// LogEventWithTimeout sends an event using a custom timeout for this call.
// Returns true if the request completed successfully with a 2xx status code.
func (s *ScarfEventLogger) LogEventWithTimeout(properties map[string]any, timeout time.Duration) bool {
    if timeout <= 0 {
        timeout = s.defaultTimeout
    }
    return s.logEventInternal(properties, timeout)
}

func (s *ScarfEventLogger) logEventInternal(properties map[string]any, timeout time.Duration) bool {
    if s.disabled {
        if s.verbose {
            s.logger.Println("analytics disabled via env; not sending event")
        }
        return false
    }

    if strings.TrimSpace(s.endpointURL) == "" {
        if s.verbose {
            s.logger.Println("no endpoint URL configured; aborting")
        }
        return false
    }

    if properties == nil {
        properties = map[string]any{}
    }

    body, err := json.Marshal(properties)
    if err != nil {
        if s.verbose {
            s.logger.Printf("failed to encode event: %v\n", err)
        }
        return false
    }

    req, err := http.NewRequest(http.MethodPost, s.endpointURL, bytes.NewReader(body))
    if err != nil {
        if s.verbose {
            s.logger.Printf("failed to build request: %v\n", err)
        }
        return false
    }
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("User-Agent", userAgent)

    // Use per-call timeout without mutating the shared client.
    client := *s.httpClient
    client.Timeout = timeout

    if s.verbose {
        s.logger.Printf("sending event to %s (timeout=%s)\n", s.endpointURL, timeout)
    }

    resp, err := client.Do(req)
    if err != nil {
        if s.verbose {
            s.logger.Printf("request failed: %v\n", err)
        }
        return false
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
        return true
    }

    if s.verbose {
        s.logger.Printf("non-success status: %s\n", resp.Status)
    }
    return false
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
        return errors.New("endpoint URL is required")
    }
    return nil
}

