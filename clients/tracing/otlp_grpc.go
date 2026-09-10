package tracing

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/encoding/gzip"

	"github.com/sven-victor/ez-utils/clients/tls"
)

// GRPCClientOptions configures an OTLP gRPC span exporter.
type GRPCClientOptions struct {
	Endpoint string        `json:"endpoint" yaml:"endpoint" mapstructure:"endpoint"`
	Timeout  time.Duration `json:"timeout" yaml:"timeout" mapstructure:"timeout"`
	// Insecure disables TLS. When false, TLSConfig is used.
	Insecure    bool              `json:"insecure" yaml:"insecure" mapstructure:"insecure"`
	Retry       RetryOptions      `json:"retry" yaml:"retry" mapstructure:"retry"`
	Compression Compression       `json:"compression" yaml:"compression" mapstructure:"compression"`
	Header      map[string]string `json:"header" yaml:"header" mapstructure:"header"`
	TLSConfig   tls.TLSOptions    `json:"tls_config" yaml:"tls_config" mapstructure:"tls_config"`
	// ReconnectionPeriod is how often the gRPC client reconnects to the endpoint.
	ReconnectionPeriod time.Duration `json:"reconnection_period" yaml:"reconnection_period" mapstructure:"reconnection_period"`
	// ServiceConfig is a gRPC service config JSON string.
	ServiceConfig string `json:"service_config" yaml:"service_config" mapstructure:"service_config"`
}

// NewGRPCTraceExporter creates an OTLP gRPC span exporter from o.
func NewGRPCTraceExporter(ctx context.Context, o *GRPCClientOptions) (sdktrace.SpanExporter, error) {
	opts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(o.Endpoint),
		otlptracegrpc.WithTimeout(o.Timeout),
		otlptracegrpc.WithRetry(otlptracegrpc.RetryConfig(o.Retry)),
		otlptracegrpc.WithHeaders(o.Header),
		otlptracegrpc.WithServiceConfig(o.ServiceConfig),
		otlptracegrpc.WithReconnectionPeriod(o.ReconnectionPeriod),
	}
	if o.Insecure {
		opts = append(opts, otlptracegrpc.WithInsecure())
	} else {
		tlsConfig, err := tls.NewTLSConfig(&o.TLSConfig)
		if err != nil {
			return nil, err
		}
		otlptracegrpc.WithTLSCredentials(credentials.NewTLS(tlsConfig))
	}
	if o.Compression == 1 {
		opts = append(opts, otlptracegrpc.WithCompressor(gzip.Name))
	}
	return otlptracegrpc.New(ctx, opts...)
}
