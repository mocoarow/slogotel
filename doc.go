// Package slogotel provides an [slog.Handler] wrapper that bridges
// Go's [log/slog] with OpenTelemetry.
//
// Unlike other slog-OpenTelemetry integrations, slogotel always injects
// trace_id and span_id as log attributes whenever a valid [trace.SpanContext]
// exists in the context, regardless of the Span's Recording state.
// When the Span is Recording, it additionally records log messages as
// Span events, injects Baggage members as log attributes, and sets
// the Span status to Error for error-level logs.
//
// # Usage
//
//	h := slogotel.New(slog.NewJSONHandler(os.Stdout, nil))
//	logger := slog.New(h)
//	logger.InfoContext(ctx, "request received", "path", "/api/users")
//
// # Options
//
// Use functional options to customize behavior:
//
//	h := slogotel.New(
//		slog.NewJSONHandler(os.Stdout, nil),
//		slogotel.WithSpanEvent(false),      // disable Span event recording
//		slogotel.WithBaggage(false),        // disable Baggage injection
//		slogotel.WithTraceIDKey("traceId"), // customize key names
//		slogotel.WithSpanIDKey("spanId"),
//	)
package slogotel
