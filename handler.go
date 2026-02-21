package slogotel

import (
	"context"
	"fmt"
	"log/slog"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Handler is a slog.Handler that integrates with OpenTelemetry.
// It adds trace_id/span_id to log records regardless of Span recording state,
// and optionally records Span events, Baggage attributes, and error status
// only when the Span is recording.
type Handler struct {
	next          slog.Handler
	addSpanEvent  bool
	addBaggage    bool
	traceIDKey    string
	spanIDKey     string
	traceFlagsKey string
}

// New creates a new Handler wrapping the given slog.Handler.
// It panics if next is nil.
func New(next slog.Handler, opts ...Option) *Handler {
	if next == nil {
		panic("slogotel: next handler must not be nil")
	}
	h := &Handler{
		next:          next,
		addSpanEvent:  true,
		addBaggage:    true,
		traceIDKey:    "trace_id",
		spanIDKey:     "span_id",
		traceFlagsKey: "",
	}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

// Enabled reports whether the handler handles records at the given level.
func (h *Handler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

// Handle processes the log record, adding OpenTelemetry context.
func (h *Handler) Handle(ctx context.Context, r slog.Record) error {
	if ctx == nil {
		ctx = context.Background()
	}

	span := trace.SpanFromContext(ctx)
	recording := span.IsRecording()

	// 1. Recording Span: add Baggage attributes (before span event capture)
	if recording && h.addBaggage {
		bag := baggage.FromContext(ctx)
		for _, m := range bag.Members() {
			r.AddAttrs(slog.String(m.Key(), m.Value()))
		}
	}

	// 2. Recording Span: record log as Span event (includes baggage, excludes trace_id/span_id)
	if recording && h.addSpanEvent {
		eventAttrs := convertAttrs(r)
		eventAttrs = append(eventAttrs, attribute.String("log.level", r.Level.String()))
		span.AddEvent(r.Message, trace.WithAttributes(eventAttrs...))
	}

	// 3. trace_id / span_id: always extract from SpanContext
	spanCtx := trace.SpanContextFromContext(ctx)
	if spanCtx.IsValid() {
		r.AddAttrs(
			slog.String(h.traceIDKey, spanCtx.TraceID().String()),
			slog.String(h.spanIDKey, spanCtx.SpanID().String()),
		)
		if h.traceFlagsKey != "" {
			r.AddAttrs(slog.String(h.traceFlagsKey, spanCtx.TraceFlags().String()))
		}
	}

	// 4. Recording Span: set error status
	if recording && r.Level >= slog.LevelError {
		span.SetStatus(codes.Error, r.Message)
	}

	if err := h.next.Handle(ctx, r); err != nil {
		return fmt.Errorf("next handler: %w", err)
	}
	return nil
}

// WithAttrs returns a new Handler with the given attributes.
func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &Handler{
		next:          h.next.WithAttrs(attrs),
		addSpanEvent:  h.addSpanEvent,
		addBaggage:    h.addBaggage,
		traceIDKey:    h.traceIDKey,
		spanIDKey:     h.spanIDKey,
		traceFlagsKey: h.traceFlagsKey,
	}
}

// WithGroup returns a new Handler with the given group name.
func (h *Handler) WithGroup(name string) slog.Handler {
	return &Handler{
		next:          h.next.WithGroup(name),
		addSpanEvent:  h.addSpanEvent,
		addBaggage:    h.addBaggage,
		traceIDKey:    h.traceIDKey,
		spanIDKey:     h.spanIDKey,
		traceFlagsKey: h.traceFlagsKey,
	}
}
