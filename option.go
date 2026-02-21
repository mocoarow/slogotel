package slogotel

// Option configures a Handler.
type Option func(*Handler)

// WithSpanEvent enables or disables recording log messages as Span events.
// Default: true.
func WithSpanEvent(enabled bool) Option {
	return func(h *Handler) {
		h.addSpanEvent = enabled
	}
}

// WithBaggage enables or disables adding Baggage members as log attributes.
// Default: true.
func WithBaggage(enabled bool) Option {
	return func(h *Handler) {
		h.addBaggage = enabled
	}
}

// WithTraceIDKey sets the key name for trace_id in log attributes.
// It panics if key is empty. Default: "trace_id".
func WithTraceIDKey(key string) Option {
	if key == "" {
		panic("slogotel: traceIDKey must not be empty")
	}
	return func(h *Handler) {
		h.traceIDKey = key
	}
}

// WithSpanIDKey sets the key name for span_id in log attributes.
// It panics if key is empty. Default: "span_id".
func WithSpanIDKey(key string) Option {
	if key == "" {
		panic("slogotel: spanIDKey must not be empty")
	}
	return func(h *Handler) {
		h.spanIDKey = key
	}
}

// WithTraceFlagsKey sets the key name for trace_flags in log attributes.
// If empty, trace_flags will not be added. Default: "" (disabled).
func WithTraceFlagsKey(key string) Option {
	return func(h *Handler) {
		h.traceFlagsKey = key
	}
}
