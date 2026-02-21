package slogotel_test

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"

	"github.com/mocoarow/slogotel"
)

func newRecordingContext(t *testing.T) (context.Context, trace.Span, *sdktrace.TracerProvider, *tracetest.InMemoryExporter) {
	t.Helper()
	tp, exporter := newTracerProvider()
	ctx, span := tp.Tracer("test").Start(context.Background(), "op")
	t.Cleanup(func() { span.End() })
	return ctx, span, tp, exporter
}

func getEventAttrs(t *testing.T, span trace.Span, tp *sdktrace.TracerProvider, exporter *tracetest.InMemoryExporter) map[string]any {
	t.Helper()
	span.End()
	require.NoError(t, tp.ForceFlush(context.Background()))
	spans := exporter.GetSpans()
	require.NotEmpty(t, spans)
	last := spans[len(spans)-1]
	require.NotEmpty(t, last.Events)
	m := make(map[string]any)
	for _, a := range last.Events[0].Attributes {
		m[string(a.Key)] = a.Value.AsInterface()
	}
	return m
}

func Test_convertAttrs_shouldConvertString(t *testing.T) {
	t.Parallel()

	// given
	ctx, span, tp, exporter := newRecordingContext(t)
	defer func() { require.NoError(t, tp.Shutdown(context.Background())) }()

	spy := newSpyHandler()
	h := slogotel.New(spy)
	r := newRecord(slog.LevelInfo, "msg")
	r.AddAttrs(slog.String("key", "value"))

	// when
	_ = h.Handle(ctx, r)

	// then
	attrs := getEventAttrs(t, span, tp, exporter)
	assert.Equal(t, "value", attrs["key"])
}

func Test_convertAttrs_shouldConvertInt64(t *testing.T) {
	t.Parallel()

	// given
	ctx, span, tp, exporter := newRecordingContext(t)
	defer func() { require.NoError(t, tp.Shutdown(context.Background())) }()

	spy := newSpyHandler()
	h := slogotel.New(spy)
	r := newRecord(slog.LevelInfo, "msg")
	r.AddAttrs(slog.Int64("count", 42))

	// when
	_ = h.Handle(ctx, r)

	// then
	attrs := getEventAttrs(t, span, tp, exporter)
	assert.Equal(t, int64(42), attrs["count"])
}

func Test_convertAttrs_shouldConvertFloat64(t *testing.T) {
	t.Parallel()

	// given
	ctx, span, tp, exporter := newRecordingContext(t)
	defer func() { require.NoError(t, tp.Shutdown(context.Background())) }()

	spy := newSpyHandler()
	h := slogotel.New(spy)
	r := newRecord(slog.LevelInfo, "msg")
	r.AddAttrs(slog.Float64("ratio", 3.14))

	// when
	_ = h.Handle(ctx, r)

	// then
	attrs := getEventAttrs(t, span, tp, exporter)
	assert.InDelta(t, 3.14, attrs["ratio"], 1e-10)
}

func Test_convertAttrs_shouldConvertBool(t *testing.T) {
	t.Parallel()

	// given
	ctx, span, tp, exporter := newRecordingContext(t)
	defer func() { require.NoError(t, tp.Shutdown(context.Background())) }()

	spy := newSpyHandler()
	h := slogotel.New(spy)
	r := newRecord(slog.LevelInfo, "msg")
	r.AddAttrs(slog.Bool("ok", true))

	// when
	_ = h.Handle(ctx, r)

	// then
	attrs := getEventAttrs(t, span, tp, exporter)
	assert.Equal(t, true, attrs["ok"])
}

func Test_convertAttrs_shouldConvertTime(t *testing.T) {
	t.Parallel()

	// given
	ctx, span, tp, exporter := newRecordingContext(t)
	defer func() { require.NoError(t, tp.Shutdown(context.Background())) }()

	spy := newSpyHandler()
	h := slogotel.New(spy)
	r := newRecord(slog.LevelInfo, "msg")
	ts := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	r.AddAttrs(slog.Time("at", ts))

	// when
	_ = h.Handle(ctx, r)

	// then
	attrs := getEventAttrs(t, span, tp, exporter)
	assert.Equal(t, ts.String(), attrs["at"])
}

func Test_convertAttrs_shouldConvertGroup(t *testing.T) {
	t.Parallel()

	// given
	ctx, span, tp, exporter := newRecordingContext(t)
	defer func() { require.NoError(t, tp.Shutdown(context.Background())) }()

	spy := newSpyHandler()
	h := slogotel.New(spy)
	r := newRecord(slog.LevelInfo, "msg")
	r.AddAttrs(slog.Group("req", slog.String("method", "GET"), slog.Int("status", 200)))

	// when
	_ = h.Handle(ctx, r)

	// then
	attrs := getEventAttrs(t, span, tp, exporter)
	assert.Equal(t, "GET", attrs["req.method"])
	assert.Equal(t, int64(200), attrs["req.status"])
}

func Test_convertAttrs_shouldConvertAny(t *testing.T) {
	t.Parallel()

	// given
	ctx, span, tp, exporter := newRecordingContext(t)
	defer func() { require.NoError(t, tp.Shutdown(context.Background())) }()

	spy := newSpyHandler()
	h := slogotel.New(spy)
	r := newRecord(slog.LevelInfo, "msg")
	custom := struct{ Name string }{"test"}
	r.AddAttrs(slog.Any("obj", custom))

	// when
	_ = h.Handle(ctx, r)

	// then
	attrs := getEventAttrs(t, span, tp, exporter)
	assert.Equal(t, fmt.Sprintf("%+v", custom), attrs["obj"])
}

// LogValuer
type testLogValuer struct {
	val string
}

func (v testLogValuer) LogValue() slog.Value {
	return slog.StringValue(v.val)
}

func Test_convertAttrs_shouldResolveLogValuer(t *testing.T) {
	t.Parallel()

	// given
	ctx, span, tp, exporter := newRecordingContext(t)
	defer func() { require.NoError(t, tp.Shutdown(context.Background())) }()

	spy := newSpyHandler()
	h := slogotel.New(spy)
	r := newRecord(slog.LevelInfo, "msg")
	r.AddAttrs(slog.Any("val", testLogValuer{val: "resolved"}))

	// when
	_ = h.Handle(ctx, r)

	// then
	attrs := getEventAttrs(t, span, tp, exporter)
	assert.Equal(t, "resolved", attrs["val"])
}

func Test_convertAttrs_shouldIncludeLogLevel(t *testing.T) {
	t.Parallel()

	// given
	ctx, span, tp, exporter := newRecordingContext(t)
	defer func() { require.NoError(t, tp.Shutdown(context.Background())) }()

	spy := newSpyHandler()
	h := slogotel.New(spy)
	r := newRecord(slog.LevelWarn, "msg")

	// when
	_ = h.Handle(ctx, r)

	// then
	attrs := getEventAttrs(t, span, tp, exporter)
	assert.Equal(t, "WARN", attrs["log.level"])
}

func Test_convertAttrs_shouldConvertUint64WithinInt64Range(t *testing.T) {
	t.Parallel()

	// given
	ctx, span, tp, exporter := newRecordingContext(t)
	defer func() { require.NoError(t, tp.Shutdown(context.Background())) }()

	spy := newSpyHandler()
	h := slogotel.New(spy)
	r := newRecord(slog.LevelInfo, "msg")
	r.AddAttrs(slog.Uint64("small", 42))

	// when
	_ = h.Handle(ctx, r)

	// then
	attrs := getEventAttrs(t, span, tp, exporter)
	assert.Equal(t, int64(42), attrs["small"])
}

func Test_convertAttrs_shouldConvertUint64ExceedingInt64AsString(t *testing.T) {
	t.Parallel()

	// given
	ctx, span, tp, exporter := newRecordingContext(t)
	defer func() { require.NoError(t, tp.Shutdown(context.Background())) }()

	spy := newSpyHandler()
	h := slogotel.New(spy)
	r := newRecord(slog.LevelInfo, "msg")
	r.AddAttrs(slog.Uint64("big", math.MaxUint64))

	// when
	_ = h.Handle(ctx, r)

	// then
	attrs := getEventAttrs(t, span, tp, exporter)
	assert.Equal(t, "18446744073709551615", attrs["big"])
}

func Test_convertAttrs_shouldConvertDuration(t *testing.T) {
	t.Parallel()

	// given
	ctx, span, tp, exporter := newRecordingContext(t)
	defer func() { require.NoError(t, tp.Shutdown(context.Background())) }()

	spy := newSpyHandler()
	h := slogotel.New(spy)
	r := newRecord(slog.LevelInfo, "msg")
	r.AddAttrs(slog.Duration("elapsed", 5*time.Second))

	// when
	_ = h.Handle(ctx, r)

	// then
	attrs := getEventAttrs(t, span, tp, exporter)
	assert.Equal(t, "5s", attrs["elapsed"])
}
