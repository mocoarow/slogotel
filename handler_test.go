package slogotel_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/mocoarow/slogotel"
)

// spyHandler captures the last Record passed to Handle.
type spyHandler struct {
	lastRecord slog.Record
	handleErr  error
	enabled    bool
	preAttrs   []slog.Attr
	group      string
}

func newSpyHandler() *spyHandler {
	return &spyHandler{enabled: true}
}

func (s *spyHandler) Enabled(_ context.Context, _ slog.Level) bool {
	return s.enabled
}

func (s *spyHandler) Handle(_ context.Context, r slog.Record) error {
	s.lastRecord = r
	return s.handleErr
}

func (s *spyHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &spyHandler{
		enabled:  s.enabled,
		preAttrs: append(s.preAttrs, attrs...),
		group:    s.group,
	}
}

func (s *spyHandler) WithGroup(name string) slog.Handler {
	return &spyHandler{
		enabled:  s.enabled,
		preAttrs: s.preAttrs,
		group:    name,
	}
}

func (s *spyHandler) attrMap() map[string]string {
	m := make(map[string]string)
	s.lastRecord.Attrs(func(a slog.Attr) bool {
		m[a.Key] = a.Value.String()
		return true
	})
	return m
}

func newTracerProvider() (*sdktrace.TracerProvider, *tracetest.InMemoryExporter) {
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
	)
	return tp, exporter
}

func newRecord(level slog.Level, msg string) slog.Record {
	return slog.NewRecord(time.Now(), level, msg, 0)
}

// --- trace_id / span_id tests ---

func Test_Handler_Handle_shouldAddTraceIDAndSpanID_whenSpanContextIsValid(t *testing.T) {
	t.Parallel()

	// given
	tp, _ := newTracerProvider()
	defer func() { require.NoError(t, tp.Shutdown(context.Background())) }()

	ctx, span := tp.Tracer("test").Start(context.Background(), "op")
	defer span.End()

	spy := newSpyHandler()
	h := slogotel.New(spy)

	// when
	err := h.Handle(ctx, slog.Record{})

	// then
	require.NoError(t, err)
	attrs := spy.attrMap()
	sc := span.SpanContext()

	assert.Equal(t, sc.TraceID().String(), attrs["trace_id"])
	assert.Equal(t, sc.SpanID().String(), attrs["span_id"])
}

func Test_Handler_Handle_shouldAddTraceIDAndSpanID_whenSpanIsNotRecording(t *testing.T) {
	t.Parallel()

	// given
	tp, _ := newTracerProvider()
	defer func() { require.NoError(t, tp.Shutdown(context.Background())) }()

	ctx, span := tp.Tracer("test").Start(context.Background(), "op")
	span.End() // End the span so it's no longer recording

	spy := newSpyHandler()
	h := slogotel.New(spy)

	// when
	err := h.Handle(ctx, slog.Record{})

	// then
	require.NoError(t, err)
	attrs := spy.attrMap()
	sc := span.SpanContext()

	assert.Equal(t, sc.TraceID().String(), attrs["trace_id"])
	assert.Equal(t, sc.SpanID().String(), attrs["span_id"])
}

func Test_Handler_Handle_shouldNotAddTraceID_whenSpanContextIsInvalid(t *testing.T) {
	t.Parallel()

	// given
	spy := newSpyHandler()
	h := slogotel.New(spy)

	// when
	err := h.Handle(context.Background(), slog.Record{})

	// then
	require.NoError(t, err)
	attrs := spy.attrMap()
	assert.NotContains(t, attrs, "trace_id")
	assert.NotContains(t, attrs, "span_id")
}

func Test_Handler_Handle_shouldNotPanic_whenContextIsNil(t *testing.T) {
	t.Parallel()

	// given
	spy := newSpyHandler()
	h := slogotel.New(spy)

	// when / then (should not panic)
	err := h.Handle(nil, slog.Record{}) //nolint:staticcheck
	require.NoError(t, err)
}

// --- Span event tests ---

func Test_Handler_Handle_shouldRecordSpanEvent_whenSpanIsRecording(t *testing.T) {
	t.Parallel()

	// given
	tp, exporter := newTracerProvider()
	defer func() { require.NoError(t, tp.Shutdown(context.Background())) }()

	ctx, span := tp.Tracer("test").Start(context.Background(), "op")

	spy := newSpyHandler()
	h := slogotel.New(spy)
	r := newRecord(slog.LevelInfo, "test message")

	// when
	_ = h.Handle(ctx, r)
	span.End()
	require.NoError(t, tp.ForceFlush(context.Background()))

	// then
	spans := exporter.GetSpans()
	require.NotEmpty(t, spans)
	require.NotEmpty(t, spans[0].Events)
	assert.Equal(t, "test message", spans[0].Events[0].Name)
}

func Test_Handler_Handle_shouldNotRecordSpanEvent_whenDisabled(t *testing.T) {
	t.Parallel()

	// given
	tp, exporter := newTracerProvider()
	defer func() { require.NoError(t, tp.Shutdown(context.Background())) }()

	ctx, span := tp.Tracer("test").Start(context.Background(), "op")

	spy := newSpyHandler()
	h := slogotel.New(spy, slogotel.WithSpanEvent(false))
	r := newRecord(slog.LevelInfo, "test message")

	// when
	_ = h.Handle(ctx, r)
	span.End()
	require.NoError(t, tp.ForceFlush(context.Background()))

	// then
	spans := exporter.GetSpans()
	require.NotEmpty(t, spans)
	assert.Empty(t, spans[0].Events)
}

func Test_Handler_Handle_shouldNotRecordSpanEvent_whenSpanIsNotRecording(t *testing.T) {
	t.Parallel()

	// given
	tp, exporter := newTracerProvider()
	defer func() { require.NoError(t, tp.Shutdown(context.Background())) }()

	ctx, span := tp.Tracer("test").Start(context.Background(), "op")
	span.End() // no longer recording

	spy := newSpyHandler()
	h := slogotel.New(spy)
	r := newRecord(slog.LevelInfo, "test message")

	// when
	_ = h.Handle(ctx, r)
	require.NoError(t, tp.ForceFlush(context.Background()))

	// then
	spans := exporter.GetSpans()
	require.NotEmpty(t, spans)
	assert.Empty(t, spans[0].Events)
}

// --- Baggage tests ---

func Test_Handler_Handle_shouldAddBaggageAttributes_whenRecording(t *testing.T) {
	t.Parallel()

	// given
	tp, _ := newTracerProvider()
	defer func() { require.NoError(t, tp.Shutdown(context.Background())) }()

	ctx, span := tp.Tracer("test").Start(context.Background(), "op")
	defer span.End()

	member, err := baggage.NewMember("tenant", "acme")
	require.NoError(t, err)
	bag, err := baggage.New(member)
	require.NoError(t, err)
	ctx = baggage.ContextWithBaggage(ctx, bag)

	spy := newSpyHandler()
	h := slogotel.New(spy)

	// when
	_ = h.Handle(ctx, slog.Record{})

	// then
	attrs := spy.attrMap()
	assert.Equal(t, "acme", attrs["tenant"])
}

func Test_Handler_Handle_shouldNotAddBaggageAttributes_whenDisabled(t *testing.T) {
	t.Parallel()

	// given
	tp, _ := newTracerProvider()
	defer func() { require.NoError(t, tp.Shutdown(context.Background())) }()

	ctx, span := tp.Tracer("test").Start(context.Background(), "op")
	defer span.End()

	member, err := baggage.NewMember("tenant", "acme")
	require.NoError(t, err)
	bag, err := baggage.New(member)
	require.NoError(t, err)
	ctx = baggage.ContextWithBaggage(ctx, bag)

	spy := newSpyHandler()
	h := slogotel.New(spy, slogotel.WithBaggage(false))

	// when
	_ = h.Handle(ctx, slog.Record{})

	// then
	attrs := spy.attrMap()
	assert.NotContains(t, attrs, "tenant")
}

func Test_Handler_Handle_shouldNotAddBaggageAttributes_whenBaggageIsEmpty(t *testing.T) {
	t.Parallel()

	// given
	tp, _ := newTracerProvider()
	defer func() { require.NoError(t, tp.Shutdown(context.Background())) }()

	ctx, span := tp.Tracer("test").Start(context.Background(), "op")
	defer span.End()

	spy := newSpyHandler()
	h := slogotel.New(spy)

	// when
	_ = h.Handle(ctx, slog.Record{})

	// then
	attrs := spy.attrMap()
	for k := range attrs {
		assert.Contains(t, []string{"trace_id", "span_id"}, k, "unexpected attribute %q", k)
	}
}

// --- Error status tests ---

func Test_Handler_Handle_shouldSetSpanStatusError_whenLevelIsError(t *testing.T) {
	t.Parallel()

	// given
	tp, exporter := newTracerProvider()
	defer func() { require.NoError(t, tp.Shutdown(context.Background())) }()

	ctx, span := tp.Tracer("test").Start(context.Background(), "op")

	spy := newSpyHandler()
	h := slogotel.New(spy)
	r := newRecord(slog.LevelError, "something failed")

	// when
	_ = h.Handle(ctx, r)
	span.End()
	require.NoError(t, tp.ForceFlush(context.Background()))

	// then
	spans := exporter.GetSpans()
	require.NotEmpty(t, spans)
	assert.Equal(t, codes.Error, spans[0].Status.Code)
	assert.Equal(t, "something failed", spans[0].Status.Description)
}

func Test_Handler_Handle_shouldNotSetSpanStatusError_whenLevelIsWarn(t *testing.T) {
	t.Parallel()

	// given
	tp, exporter := newTracerProvider()
	defer func() { require.NoError(t, tp.Shutdown(context.Background())) }()

	ctx, span := tp.Tracer("test").Start(context.Background(), "op")

	spy := newSpyHandler()
	h := slogotel.New(spy)
	r := newRecord(slog.LevelWarn, "warning")

	// when
	_ = h.Handle(ctx, r)
	span.End()
	require.NoError(t, tp.ForceFlush(context.Background()))

	// then
	spans := exporter.GetSpans()
	require.NotEmpty(t, spans)
	assert.NotEqual(t, codes.Error, spans[0].Status.Code)
}

// --- Edge case tests ---

func Test_Handler_Handle_shouldPropagateNextHandlerError(t *testing.T) {
	t.Parallel()

	// given
	spy := &spyHandler{enabled: true, handleErr: errors.New("next handler error")}
	h := slogotel.New(spy)

	// when
	err := h.Handle(context.Background(), slog.Record{})

	// then
	require.EqualError(t, err, "next handler: next handler error")
}

func Test_Handler_Handle_shouldWorkWithContextBackground(t *testing.T) {
	t.Parallel()

	// given
	spy := newSpyHandler()
	h := slogotel.New(spy)

	// when
	err := h.Handle(context.Background(), slog.Record{})

	// then
	require.NoError(t, err)
	attrs := spy.attrMap()
	assert.Empty(t, attrs)
}

func Test_Handler_Enabled_shouldDelegateToNext(t *testing.T) {
	t.Parallel()

	// given
	spy := &spyHandler{enabled: false}
	h := slogotel.New(spy)

	// when / then
	assert.False(t, h.Enabled(context.Background(), slog.LevelInfo))
}

func Test_New_shouldPanic_whenNextIsNil(t *testing.T) {
	t.Parallel()

	// given / when / then
	assert.PanicsWithValue(t, "slogotel: next handler must not be nil", func() {
		slogotel.New(nil)
	})
}

func Test_Handler_Handle_shouldNotIncludeTraceIDInSpanEvent(t *testing.T) {
	t.Parallel()

	// given
	tp, exporter := newTracerProvider()
	defer func() { require.NoError(t, tp.Shutdown(context.Background())) }()

	ctx, span := tp.Tracer("test").Start(context.Background(), "op")

	spy := newSpyHandler()
	h := slogotel.New(spy)
	r := newRecord(slog.LevelInfo, "msg")
	r.AddAttrs(slog.String("user", "alice"))

	// when
	_ = h.Handle(ctx, r)
	span.End()
	require.NoError(t, tp.ForceFlush(context.Background()))

	// then
	spans := exporter.GetSpans()
	require.NotEmpty(t, spans)
	require.NotEmpty(t, spans[0].Events)
	for _, attr := range spans[0].Events[0].Attributes {
		key := string(attr.Key)
		assert.NotEqual(t, "trace_id", key, "span event should not contain trace_id")
		assert.NotEqual(t, "span_id", key, "span event should not contain span_id")
	}
}

// --- WithAttrs / WithGroup tests ---

func Test_Handler_WithAttrs_shouldReturnNewHandler(t *testing.T) {
	t.Parallel()

	// given
	tp, _ := newTracerProvider()
	defer func() { require.NoError(t, tp.Shutdown(context.Background())) }()

	ctx, span := tp.Tracer("test").Start(context.Background(), "op")
	defer span.End()

	spy := newSpyHandler()
	h := slogotel.New(spy)

	// when
	h2 := h.WithAttrs([]slog.Attr{slog.String("env", "prod")})

	// then
	assert.NotEqual(t, h, h2, "WithAttrs should return a new handler")

	err := h2.Handle(ctx, slog.Record{})
	require.NoError(t, err)
}

func Test_Handler_WithGroup_shouldReturnNewHandler(t *testing.T) {
	t.Parallel()

	// given
	tp, _ := newTracerProvider()
	defer func() { require.NoError(t, tp.Shutdown(context.Background())) }()

	ctx, span := tp.Tracer("test").Start(context.Background(), "op")
	defer span.End()

	spy := newSpyHandler()
	h := slogotel.New(spy)

	// when
	h2 := h.WithGroup("request")

	// then
	assert.NotEqual(t, h, h2, "WithGroup should return a new handler")

	err := h2.Handle(ctx, slog.Record{})
	require.NoError(t, err)
}
