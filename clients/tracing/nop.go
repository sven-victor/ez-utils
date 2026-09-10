package tracing

import (
	"context"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

type nopTraceExporter struct{}

func (n nopTraceExporter) ExportSpans(ctx context.Context, spans []sdktrace.ReadOnlySpan) error {
	return nil
}

func (n nopTraceExporter) Shutdown(ctx context.Context) error {
	return nil
}

// NewNopTraceExporter returns a span exporter that discards all spans.
func NewNopTraceExporter(_ context.Context) (sdktrace.SpanExporter, error) {
	return &nopTraceExporter{}, nil
}
