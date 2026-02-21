package slogotel

import (
	"fmt"
	"log/slog"
	"math"
	"strconv"

	"go.opentelemetry.io/otel/attribute"
)

const maxGroupDepth = 10

// convertAttrs converts slog.Record attributes to OTel attributes.
func convertAttrs(r slog.Record) []attribute.KeyValue {
	attrs := make([]attribute.KeyValue, 0, r.NumAttrs())
	r.Attrs(func(a slog.Attr) bool {
		attrs = append(attrs, convertAttr("", a, 0)...)
		return true
	})
	return attrs
}

// convertAttr converts a single slog.Attr to OTel attributes.
// For KindUint64, values within int64 range are stored as Int64;
// values exceeding math.MaxInt64 are converted to their decimal string representation.
func convertAttr(prefix string, a slog.Attr, depth int) []attribute.KeyValue {
	a.Value = a.Value.Resolve()
	key := a.Key
	if prefix != "" {
		key = prefix + "." + key
	}

	switch a.Value.Kind() {
	case slog.KindString:
		return []attribute.KeyValue{attribute.String(key, a.Value.String())}
	case slog.KindInt64:
		return []attribute.KeyValue{attribute.Int64(key, a.Value.Int64())}
	case slog.KindUint64:
		v := a.Value.Uint64()
		if v <= uint64(math.MaxInt64) {
			return []attribute.KeyValue{attribute.Int64(key, int64(v))}
		}
		return []attribute.KeyValue{attribute.String(key, strconv.FormatUint(v, 10))}
	case slog.KindFloat64:
		return []attribute.KeyValue{attribute.Float64(key, a.Value.Float64())}
	case slog.KindBool:
		return []attribute.KeyValue{attribute.Bool(key, a.Value.Bool())}
	case slog.KindTime:
		return []attribute.KeyValue{attribute.String(key, a.Value.Time().String())}
	case slog.KindDuration:
		return []attribute.KeyValue{attribute.String(key, a.Value.Duration().String())}
	case slog.KindGroup:
		return convertGroup(key, a.Value.Group(), depth)
	case slog.KindAny, slog.KindLogValuer:
		return []attribute.KeyValue{attribute.String(key, fmt.Sprintf("%+v", a.Value.Any()))}
	}

	return []attribute.KeyValue{attribute.String(key, fmt.Sprintf("%+v", a.Value.Any()))}
}

func convertGroup(prefix string, attrs []slog.Attr, depth int) []attribute.KeyValue {
	if depth >= maxGroupDepth {
		return []attribute.KeyValue{attribute.String(prefix, fmt.Sprintf("%+v", attrs))}
	}
	var result []attribute.KeyValue
	for _, a := range attrs {
		result = append(result, convertAttr(prefix, a, depth+1)...)
	}
	return result
}
