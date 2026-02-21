package slogotel_test

import (
	"log/slog"
	"os"

	"github.com/mocoarow/slogotel"
)

func ExampleNew() {
	h := slogotel.New(slog.NewJSONHandler(os.Stdout, nil))
	logger := slog.New(h)
	logger.Info("hello", "key", "value")
}

func ExampleNew_withOptions() {
	h := slogotel.New(
		slog.NewJSONHandler(os.Stdout, nil),
		slogotel.WithSpanEvent(false),
		slogotel.WithBaggage(false),
		slogotel.WithTraceIDKey("traceId"),
		slogotel.WithSpanIDKey("spanId"),
		slogotel.WithTraceFlagsKey("traceFlags"),
	)
	logger := slog.New(h)
	logger.Info("hello", "key", "value")
}
