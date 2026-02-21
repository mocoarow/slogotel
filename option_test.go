package slogotel_test

import (
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mocoarow/slogotel"
)

func Test_New_shouldSetDefaults(t *testing.T) {
	t.Parallel()

	// given
	tp, _ := newTracerProvider()
	defer func() { require.NoError(t, tp.Shutdown(context.Background())) }()

	ctx, span := tp.Tracer("test").Start(context.Background(), "op")
	defer span.End()

	spy := newSpyHandler()

	// when
	h := slogotel.New(spy)
	err := h.Handle(ctx, slog.Record{})

	// then
	require.NoError(t, err)
	attrs := spy.attrMap()
	assert.Contains(t, attrs, "trace_id")
	assert.Contains(t, attrs, "span_id")
}

func Test_WithTraceIDKey_shouldCustomizeKey(t *testing.T) {
	t.Parallel()

	// given
	tp, _ := newTracerProvider()
	defer func() { require.NoError(t, tp.Shutdown(context.Background())) }()

	ctx, span := tp.Tracer("test").Start(context.Background(), "op")
	defer span.End()

	spy := newSpyHandler()
	h := slogotel.New(spy, slogotel.WithTraceIDKey("traceId"))

	// when
	_ = h.Handle(ctx, slog.Record{})

	// then
	attrs := spy.attrMap()
	assert.Contains(t, attrs, "traceId")
	assert.NotContains(t, attrs, "trace_id")
}

func Test_WithSpanIDKey_shouldCustomizeKey(t *testing.T) {
	t.Parallel()

	// given
	tp, _ := newTracerProvider()
	defer func() { require.NoError(t, tp.Shutdown(context.Background())) }()

	ctx, span := tp.Tracer("test").Start(context.Background(), "op")
	defer span.End()

	spy := newSpyHandler()
	h := slogotel.New(spy, slogotel.WithSpanIDKey("spanId"))

	// when
	_ = h.Handle(ctx, slog.Record{})

	// then
	attrs := spy.attrMap()
	assert.Contains(t, attrs, "spanId")
	assert.NotContains(t, attrs, "span_id")
}

func Test_WithTraceFlagsKey_shouldAddTraceFlags(t *testing.T) {
	t.Parallel()

	// given
	tp, _ := newTracerProvider()
	defer func() { require.NoError(t, tp.Shutdown(context.Background())) }()

	ctx, span := tp.Tracer("test").Start(context.Background(), "op")
	defer span.End()

	spy := newSpyHandler()
	h := slogotel.New(spy, slogotel.WithTraceFlagsKey("trace_flags"))

	// when
	_ = h.Handle(ctx, slog.Record{})

	// then
	attrs := spy.attrMap()
	assert.Contains(t, attrs, "trace_flags")
}

func Test_WithTraceFlagsKey_shouldNotAddTraceFlags_whenEmpty(t *testing.T) {
	t.Parallel()

	// given
	tp, _ := newTracerProvider()
	defer func() { require.NoError(t, tp.Shutdown(context.Background())) }()

	ctx, span := tp.Tracer("test").Start(context.Background(), "op")
	defer span.End()

	spy := newSpyHandler()
	h := slogotel.New(spy) // default: empty

	// when
	_ = h.Handle(ctx, slog.Record{})

	// then
	attrs := spy.attrMap()
	assert.NotContains(t, attrs, "trace_flags")
}

func Test_WithTraceIDKey_shouldPanic_whenEmpty(t *testing.T) {
	t.Parallel()

	// given / when / then
	assert.PanicsWithValue(t, "slogotel: traceIDKey must not be empty", func() {
		slogotel.WithTraceIDKey("")
	})
}

func Test_WithSpanIDKey_shouldPanic_whenEmpty(t *testing.T) {
	t.Parallel()

	// given / when / then
	assert.PanicsWithValue(t, "slogotel: spanIDKey must not be empty", func() {
		slogotel.WithSpanIDKey("")
	})
}
