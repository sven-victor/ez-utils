package tracing

import (
	"context"
	"os"

	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// FileTracingOptions writes spans as pretty-printed JSON to Path.
type FileTracingOptions struct {
	Path string `json:"path" yaml:"path" mapstructure:"path"`
}

// FileTracing is a stdout span exporter that writes to a file.
type FileTracing struct {
	*stdouttrace.Exporter
	f *os.File
}

// Shutdown flushes and closes the output file.
func (t *FileTracing) Shutdown(ctx context.Context) error {
	if err := t.f.Sync(); err != nil {
		return err
	}
	return t.f.Close()
}

// NewFileTraceExporter creates a span exporter that writes pretty-printed traces to o.Path.
func NewFileTraceExporter(_ context.Context, o *FileTracingOptions) (sdktrace.SpanExporter, error) {
	f, err := os.Create(o.Path)
	if err != nil {
		return nil, err
	}
	exporter, err := stdouttrace.New(
		stdouttrace.WithWriter(f),
		stdouttrace.WithPrettyPrint(),
	)
	if err != nil {
		return nil, err
	}

	return &FileTracing{
		Exporter: exporter,
		f:        f,
	}, nil
}
