package scarf

import (
    "fmt"
    "net/http"
    "net/http/httptest"
    "os"
    "encoding/json"
    "strings"
    "testing"
    "time"
)

func TestDisabledViaEnv(t *testing.T) {
    t.Setenv("DO_NOT_TRACK", "1")
    logger := NewScarfEventLogger("https://example.com")
    if logger.Enabled() {
        t.Fatalf("expected logger to be disabled via env")
    }
    if err := logger.LogEvent(map[string]any{"event": "test"}); err == nil {
        t.Fatalf("expected LogEvent to return error when disabled")
    }
}

func TestLogEvent_Success(t *testing.T) {
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodPost {
            t.Fatalf("expected POST, got %s", r.Method)
        }
        if ua := r.Header.Get("User-Agent"); !strings.HasPrefix(ua, "scarf-go/") || !strings.Contains(ua, "(platform=") || !strings.Contains(ua, "arch=") || !strings.Contains(ua, "go=") {
            t.Fatalf("unexpected user-agent format: %q", ua)
        }
        // Validate query params are populated
        q := r.URL.Query()
        if ev := q.Get("event"); ev == "" {
            t.Fatalf("expected 'event' query param to be set")
        }
        w.WriteHeader(http.StatusOK)
    }))
    defer srv.Close()

    // Ensure verbose does not break anything
    os.Setenv("SCARF_VERBOSE", "1")
    t.Cleanup(func() { os.Unsetenv("SCARF_VERBOSE") })

    l := NewScarfEventLogger(srv.URL, 2*time.Second)
    if err := l.LogEvent(map[string]any{"event": "ok"}); err != nil {
        t.Fatalf("expected LogEvent to succeed, got %v", err)
    }

    if err := l.LogEventWithTimeout(map[string]any{"event": "ok2"}, 1*time.Second); err != nil {
        t.Fatalf("expected LogEventWithTimeout to succeed, got %v", err)
    }
}

func TestLogEvent_InvalidEndpoint(t *testing.T) {
    l := NewScarfEventLogger("", 1*time.Second)
    if err := l.LogEvent(map[string]any{"event": "bad"}); err == nil {
        t.Fatalf("expected failure with empty endpoint URL")
    }
}

type stringerType int
func (s stringerType) String() string { return fmt.Sprintf("stringer-%d", int(s)) }

func TestLogEvent_QueryEncoding(t *testing.T) {

    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodPost {
            t.Fatalf("expected POST, got %s", r.Method)
        }
        q := r.URL.Query()
        if got := q.Get("s"); got != "hello" {
            t.Fatalf("expected s=hello, got %q", got)
        }
        if got := q.Get("i"); got != "42" {
            t.Fatalf("expected i=42, got %q", got)
        }
        if got := q.Get("b"); got != "true" {
            t.Fatalf("expected b=true, got %q", got)
        }
        // Arrays encoded as JSON
        var arr []int
        if err := json.Unmarshal([]byte(q.Get("arr")), &arr); err != nil || len(arr) != 3 || arr[0] != 1 || arr[1] != 2 || arr[2] != 3 {
            t.Fatalf("expected arr=[1,2,3], got %q (err=%v)", q.Get("arr"), err)
        }
        // Objects encoded as JSON
        var obj map[string]string
        if err := json.Unmarshal([]byte(q.Get("obj")), &obj); err != nil || obj["a"] != "b" {
            t.Fatalf("expected obj={\"a\":\"b\"}, got %q (err=%v)", q.Get("obj"), err)
        }
        // Stringer uses its String()
        if got := q.Get("str"); !strings.HasPrefix(got, "stringer-") {
            t.Fatalf("expected str to start with stringer-, got %q", got)
        }
        w.WriteHeader(http.StatusOK)
    }))
    defer srv.Close()

    l := NewScarfEventLogger(srv.URL)
    err := l.LogEvent(map[string]any{
        "s":   "hello",
        "i":   42,
        "b":   true,
        "arr": []int{1, 2, 3},
        "obj": map[string]string{"a": "b"},
        "str": stringerType(1),
    })
    if err != nil {
        t.Fatalf("expected success, got %v", err)
    }
}

func TestLogEvent_NonSuccessStatus(t *testing.T) {
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusInternalServerError)
    }))
    defer srv.Close()

    l := NewScarfEventLogger(srv.URL)
    if err := l.LogEvent(map[string]any{"event": "boom"}); err == nil {
        t.Fatalf("expected error on non-2xx status")
    }
}
