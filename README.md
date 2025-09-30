Scarf Go SDK

Simple telemetry event logger for Scarf (scarf.sh) endpoints.

Usage

```go
package main

import (
    "time"
    "github.com/scarf-sh/scarf-go/scarf"
)

func main() {
    // Initialize with required endpoint URL
    logger := scarf.NewScarfEventLogger("https://your-scarf-endpoint.com")

    // Optional: set a default timeout (default is 3 seconds)
    logger = scarf.NewScarfEventLogger("https://your-scarf-endpoint.com", 5*time.Second)

    // Send an event with properties
    ok := logger.LogEvent(map[string]any{
        "event":   "package_download",
        "package": "scarf",
        "version": "1.0.0",
    })
    _ = ok

    // Send an event with a custom timeout overriding the default
    ok = logger.LogEventWithTimeout(map[string]any{"event": "custom_event"}, 1*time.Second)
    _ = ok

    // Empty properties are allowed
    ok = logger.LogEvent(map[string]any{})
    _ = ok
}
```

Configuration

The client can be configured through environment variables:

- `DO_NOT_TRACK=1`: Disable analytics
- `SCARF_NO_ANALYTICS=1`: Disable analytics (alternative)
- `SCARF_VERBOSE=1`: Enable verbose logging

Features

- Simple API for sending telemetry events
- Environment variable configuration
- Configurable timeouts (default: 3 seconds)
- Respects user Do Not Track settings
- Verbose logging mode for debugging

Notes

- This package uses only the Go standard library, no external dependencies.
- `LogEvent` returns `true` when the HTTP request is successfully sent and receives a 2xx status code. When analytics are disabled via env vars, it returns `false` and does not send a request.
