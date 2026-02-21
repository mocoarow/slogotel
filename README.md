# slogotel

An `slog.Handler` wrapper library that bridges Go's `log/slog` with OpenTelemetry.

## Features

- **Always injects trace_id / span_id** — Adds trace context as log attributes whenever a valid SpanContext exists, regardless of the Span's Recording state
- **Span event recording** — Records log messages as Span events when a Recording Span is present
- **Baggage injection** — Adds Baggage members as log attributes when a Recording Span is present
- **Error Span status** — Sets Span Status to Error when log level is `slog.LevelError` or above and the Span is Recording
- **Functional Options** — Enable/disable each feature and customize key names
- **slogtest.TestHandler compliant** — Passes the standard conformance tests

## Installation

```sh
go get github.com/mocoarow/slogotel
```

## Usage

### Basic

```go
package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/mocoarow/slogotel"
)

func main() {
	ctx := context.Background()

	h := slogotel.New(slog.NewJSONHandler(os.Stdout, nil))
	logger := slog.New(h)

	// trace_id and span_id are automatically injected if a Span exists in the context
	logger.InfoContext(ctx, "request received", "path", "/api/users")
}
```

Output example:

```json
{
  "time": "2024-01-01T00:00:00Z",
  "level": "INFO",
  "msg": "request received",
  "trace_id": "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4",
  "span_id": "a1b2c3d4e5f6a1b2",
  "path": "/api/users"
}
```

### With Options

```go
h := slogotel.New(
	slog.NewJSONHandler(os.Stdout, nil),
	slogotel.WithSpanEvent(false),       // Disable Span event recording
	slogotel.WithBaggage(false),         // Disable Baggage injection
	slogotel.WithTraceIDKey("traceId"),  // Customize key names
	slogotel.WithSpanIDKey("spanId"),
	slogotel.WithTraceFlagsKey("traceFlags"),
)
```

## Options

| Option | Default | Description |
|--------|---------|-------------|
| `WithSpanEvent(bool)` | `true` | Record log messages as Span events on Recording Spans |
| `WithBaggage(bool)` | `true` | Add Baggage members as log attributes on Recording Spans |
| `WithTraceIDKey(string)` | `"trace_id"` | Key name for the trace_id attribute |
| `WithSpanIDKey(string)` | `"span_id"` | Key name for the span_id attribute |
| `WithTraceFlagsKey(string)` | `""` (disabled) | Key name for the trace_flags attribute (empty string disables it) |

## Processing Flow

```
Handle(ctx, record)
  │
  ├── 1. Get SpanContext ← Independent of Recording state
  │     ├── trace_id is Valid → Add attribute
  │     └── span_id is Valid → Add attribute
  │
  ├── 2. Only when Span is Recording:
  │     ├── Add Baggage members as log attributes
  │     ├── Record log as a Span event
  │     └── Set Span Status to Error if level ≥ LevelError
  │
  └── 3. next.Handle(ctx, record)
```

## Notes

- If a Baggage key conflicts with an existing log attribute key, both values are appended to the record (the downstream handler, such as `slog.JSONHandler`, typically uses the last value)
- The `log.level` attribute is automatically added to Span events
- Group attributes are flattened with dot-separated keys (e.g., `req.method`)

## License

MIT
