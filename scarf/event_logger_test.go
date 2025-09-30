package scarf

import (
    "net/http"
    "net/http/httptest"
    "os"
    "testing"
    "time"
)

func TestDisabledViaEnv(t *testing.T) {
    t.Setenv("DO_NOT_TRACK", "1")
    logger := NewScarfEventLogger("https://example.com")
    if logger.Enabled() {
        t.Fatalf("expected logger to be disabled via env")
    }
    if ok := logger.LogEvent(map[string]any{"event": "test"}); ok {
        t.Fatalf("expected LogEvent to return false when disabled")
    }
}

func TestLogEvent_Success(t *testing.T) {
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodPost {
            t.Fatalf("expected POST, got %s", r.Method)
        }
        if ct := r.Header.Get("Content-Type"); ct != "application/json" {
            t.Fatalf("expected content-type application/json, got %s", ct)
        }
        w.WriteHeader(http.StatusOK)
    }))
    defer srv.Close()

    // Ensure verbose does not break anything
    os.Setenv("SCARF_VERBOSE", "1")
    t.Cleanup(func() { os.Unsetenv("SCARF_VERBOSE") })

    l := NewScarfEventLogger(srv.URL, 2*time.Second)
    ok := l.LogEvent(map[string]any{"event": "ok"})
    if !ok {
        t.Fatalf("expected LogEvent to succeed")
    }

    ok = l.LogEventWithTimeout(map[string]any{"event": "ok2"}, 1*time.Second)
    if !ok {
        t.Fatalf("expected LogEventWithTimeout to succeed")
    }
}

func TestLogEvent_InvalidEndpoint(t *testing.T) {
    l := NewScarfEventLogger("", 1*time.Second)
    ok := l.LogEvent(map[string]any{"event": "bad"})
    if ok {
        t.Fatalf("expected failure with empty endpoint URL")
    }
}

