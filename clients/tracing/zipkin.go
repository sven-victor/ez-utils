package tracing

import (
	"context"

	"go.opentelemetry.io/otel/exporters/zipkin"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// ZipkinClientOptions configures a Zipkin span exporter.
type ZipkinClientOptions struct {
	Endpoint string `json:"endpoint" yaml:"endpoint" mapstructure:"endpoint"`
}

// NewZipkinTraceExporter creates a Zipkin span exporter that posts to o.Endpoint.
func NewZipkinTraceExporter(_ context.Context, o *ZipkinClientOptions) (sdktrace.SpanExporter, error) {
	return zipkin.New(o.Endpoint)
}
