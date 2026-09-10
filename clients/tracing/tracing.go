// Package tracing configures OpenTelemetry tracer providers and span exporters
// for OTLP HTTP, OTLP gRPC, Zipkin, file, and no-op backends.
package tracing

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"time"

	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"
	uuid "github.com/satori/go.uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/sven-victor/ez-utils/log"
	"github.com/sven-victor/ez-utils/signals"
)

// RetryOptions controls exporter retry backoff. UnmarshalJSON accepts false to
// disable retries, or a JSON object; omitted fields use DefaultRetryConfig.
type RetryOptions struct {
	Enabled bool
	// InitialInterval is the first retry backoff.
	InitialInterval time.Duration
	// MaxInterval caps the backoff between retries.
	MaxInterval time.Duration
	// MaxElapsedTime stops retrying after this duration.
	MaxElapsedTime time.Duration
}

// UnmarshalJSON decodes retry settings. The JSON values false or "false"
// disable retries; otherwise defaults are applied before overlaying the object.
func (c *RetryOptions) UnmarshalJSON(data []byte) (err error) {
	type plain RetryOptions
	*c = RetryOptions{
		Enabled:         true,
		InitialInterval: 5 * time.Second,
		MaxInterval:     30 * time.Second,
		MaxElapsedTime:  time.Minute,
	}
	if string(data) == "false" || string(data) == `"false"` {
		c.Enabled = false
		return
	}
	return json.Unmarshal(data, (*plain)(c))
}

// DefaultRetryConfig is the retry policy used when Retry is omitted from JSON.
var DefaultRetryConfig = RetryOptions{
	Enabled:         true,
	InitialInterval: 5 * time.Second,
	MaxInterval:     30 * time.Second,
	MaxElapsedTime:  time.Minute,
}

// Compression selects OTLP payload compression. JSON true, 1, or "gzip" enable
// gzip; false, 0, or empty disable it.
type Compression int

// UnmarshalJSON sets gzip or no compression from a JSON boolean, number, or string.
func (c *Compression) UnmarshalJSON(data []byte) (err error) {
	switch string(data) {
	case `"true"`, `true`, `1`, `"1"`, `"gzip"`:
		*c = Compression(otlptracehttp.GzipCompression)
	case `"false"`, `false`, `0`, `"0"`, ``:
		*c = Compression(otlptracehttp.NoCompression)
	}
	return fmt.Errorf("the value can only be one of true, false, or gzip")
}

// TraceOptions selects a single span exporter backend and the service name
// recorded on the OpenTelemetry resource. Set exactly one of HTTP, GRPC,
// Zipkin, or File; if none is set, NewTraceProvider uses a no-op exporter.
type TraceOptions struct {
	HTTP        *HTTPClientOptions   `json:"http" yaml:"http" mapstructure:"http"`
	GRPC        *GRPCClientOptions   `json:"grpc" yaml:"grpc" mapstructure:"grpc"`
	Zipkin      *ZipkinClientOptions `json:"zipkin" yaml:"zipkin" mapstructure:"zipkin"`
	File        *FileTracingOptions  `json:"file" yaml:"file" mapstructure:"file"`
	ServiceName string               `json:"service_name" yaml:"service_name" mapstructure:"service_name"`
}

// UnmarshalJSON decodes TraceOptions from JSON.
func (o *TraceOptions) UnmarshalJSON(data []byte) (err error) {
	type plain TraceOptions
	return json.Unmarshal(data, (*plain)(o))
}

// String implements proto.Message.
func (o TraceOptions) String() string {
	return proto.CompactTextString(&o)
}

// ProtoMessage implements proto.Message.
func (o *TraceOptions) ProtoMessage() {}

// Reset implements proto.Message.
func (o *TraceOptions) Reset() {
	*o = TraceOptions{}
}

// MarshalJSONPB encodes the selected backend for protobuf JSON.
func (o *TraceOptions) MarshalJSONPB(marshaller *jsonpb.Marshaler) ([]byte, error) {
	buf := &bytes.Buffer{}
	buf.WriteString(marshaller.Indent + "{")
	enc := json.NewEncoder(buf)
	enc.SetIndent(marshaller.Indent, marshaller.Indent)
	var err error
	if o.HTTP != nil {
		buf.WriteString(marshaller.Indent + marshaller.Indent + `"http":`)
		err = enc.Encode(o.HTTP)
		buf.WriteRune('\n')
	} else if o.GRPC != nil {
		buf.WriteString(marshaller.Indent + marshaller.Indent + `"grpc":`)
		err = enc.Encode(o.GRPC)
		buf.WriteRune('\n')
	} else if o.Zipkin != nil {
		buf.WriteString(marshaller.Indent + marshaller.Indent + `"zipkin":`)
		err = enc.Encode(o.Zipkin)
		buf.WriteRune('\n')
	} else if o.File != nil {
		buf.WriteString(marshaller.Indent + marshaller.Indent + `"file":`)
		err = enc.Encode(o.File)
		buf.WriteRune('\n')
	}
	buf.WriteString(marshaller.Indent + "}")
	return buf.Bytes(), err
}

type idGenerator struct{}

func (i idGenerator) NewIDs(ctx context.Context) (trace.TraceID, trace.SpanID) {
	tid, err := uuid.FromString(log.GetTraceId(ctx))
	if err != nil {
		tid = uuid.Must(uuid.NewV4())
	}

	sid := trace.SpanID{}
	_, _ = rand.Read(sid[:])
	return trace.TraceID(tid), sid
}

func (i idGenerator) NewSpanID(ctx context.Context, traceID trace.TraceID) trace.SpanID {
	sid := trace.SpanID{}
	_, _ = rand.Read(sid[:])
	return sid
}

// DefaultOptions is the process-wide tracing configuration set by SetTraceOptions.
var DefaultOptions *TraceOptions

// SetTraceOptions stores o as DefaultOptions.
func SetTraceOptions(o *TraceOptions) {
	DefaultOptions = o
}

// TraceProviderOptions holds extra resource attributes for NewTraceProvider.
type TraceProviderOptions struct {
	Attrs []attribute.KeyValue
}

// WithAttributes appends resource attributes when constructing a TracerProvider.
func WithAttributes(attrs ...attribute.KeyValue) func(*TraceProviderOptions) {
	return func(o *TraceProviderOptions) {
		o.Attrs = append(o.Attrs, attrs...)
	}
}

// NewTraceProviderOption configures NewTraceProvider.
type NewTraceProviderOption func(*TraceProviderOptions)

// NewTraceProvider builds a TracerProvider from o. It selects an exporter from
// HTTP, GRPC, Zipkin, or File, or a no-op exporter if none is set. The process
// signal handler flushes and shuts down the provider on shutdown.
func NewTraceProvider(ctx context.Context, o *TraceOptions, opts ...NewTraceProviderOption) (p *sdktrace.TracerProvider, err error) {
	var options TraceProviderOptions
	for _, opt := range opts {
		opt(&options)
	}
	var logger kitlog.Logger
	ctx, logger = log.NewContextLogger(ctx)
	var exp sdktrace.SpanExporter
	if o.HTTP != nil {
		exp, err = NewHTTPTraceExporter(ctx, o.HTTP)
	} else if o.GRPC != nil {
		exp, err = NewGRPCTraceExporter(ctx, o.GRPC)
	} else if o.Zipkin != nil {
		exp, err = NewZipkinTraceExporter(ctx, o.Zipkin)
	} else if o.File != nil {
		exp, err = NewFileTraceExporter(ctx, o.File)
	} else {
		exp, err = NewNopTraceExporter(ctx)
	}
	if err != nil {
		return nil, err
	}
	var attrs []attribute.KeyValue
	if len(o.ServiceName) != 0 {
		attrs = append(attrs, semconv.ServiceName(o.ServiceName))
	}
	if ns := os.Getenv("OTEL_SERVICE_NAMESPACE"); len(ns) != 0 {
		attrs = append(attrs, semconv.ServiceNamespace(ns))
	}
	attrs = append(attrs, semconv.HostArchKey.String(runtime.GOARCH))

	r, err := resource.New(ctx,
		resource.WithContainer(),
		resource.WithOS(),
		resource.WithHost(),
		resource.WithProcess(),
		resource.WithAttributes(attrs...),
		resource.WithAttributes(options.Attrs...),
	)
	if err != nil {
		return nil, err
	}

	r, err = resource.Merge(
		resource.Default(),
		r,
	)
	if err != nil {
		return nil, err
	}

	otel.SetTextMapPropagator(propagation.TraceContext{})
	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) {
		level.Error(logger).Log("msg", "trace exception", "err", err)
	}))
	tracer := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(r),
		sdktrace.WithIDGenerator(&idGenerator{}),
	)
	stopCh := signals.SignalHandler()
	stopCh.PreStop(signals.LevelTrace, func() {
		if tracer != nil {
			timeoutCtx, closeCh := context.WithTimeout(context.Background(), time.Second*5)
			defer closeCh()
			if err = tracer.ForceFlush(timeoutCtx); err != nil {
				level.Error(logger).Log("msg", "failed to force flush trace", "err", err)
			}
			if err = tracer.Shutdown(timeoutCtx); err != nil {
				level.Error(logger).Log("msg", "failed to close trace", "err", err)
			}
		}
	})
	return tracer, nil
}
