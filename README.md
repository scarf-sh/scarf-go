# Scarf Go SDK

Simple telemetry event logger for [Scarf](https://scarf.sh) endpoints.

## Usage

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
    if err := logger.LogEvent(map[string]any{
        "event":   "package_download",
        "package": "scarf",
        "version": "1.0.0",
    }); err != nil {
        // handle error
    }

    // Send an event with a custom timeout overriding the default
    if err := logger.LogEventWithTimeout(map[string]any{"event": "custom_event"}, 1*time.Second); err != nil {
        // handle error
    }

    // Empty properties are allowed
    if err := logger.LogEvent(map[string]any{}); err != nil {
        // handle error
    }
}
```

## Configuration

The client can be configured through environment variables:

- `DO_NOT_TRACK=1`: Disable analytics
- `SCARF_NO_ANALYTICS=1`: Disable analytics (alternative)
- `SCARF_VERBOSE=1`: Enable verbose logging

## Features

- Simple API for sending telemetry events
- Environment variable configuration
- Configurable timeouts (default: 3 seconds)
- Respects user Do Not Track settings
- Verbose logging mode for debugging

## Notes

- `LogEvent` returns `nil` when the HTTP request is successfully sent and receives a 2xx status code. When analytics are disabled via env vars, it returns a non-nil error.
- The `User-Agent` includes SDK version, platform, architecture, and Go version in a self-describing format (e.g., `scarf-go/v1.2.3 (os=macOS; arch=arm64; go=1.22.3)`).
- This package uses only the Go standard library, no external dependencies.

## Request format

- Events are sent as `POST` requests, with all provided properties encoded as URL query parameters on the endpoint URL. No JSON body is sent.

## License

- Licensed under the Apache License, Version 2.0. See `LICENSE` for details.
