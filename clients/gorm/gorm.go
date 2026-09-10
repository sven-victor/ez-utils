// Package gorm provides GORM clients for MySQL, SQLite, and ClickHouse with
// structured logging, OpenTelemetry tracing, and Prometheus connection metrics.
package gorm

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-kit/log/level"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/gorm"

	logs "github.com/sven-victor/ez-utils/log"
)

// DBOptions describes a database connection used for tracing attributes,
// metrics labels, and log messages.
type DBOptions interface {
	GetConnectionString() string
	GetPeer() (string, int)
	GetDBName() string
	GetUsername() string
	GetType() string
	String() string
}

// SessionClient is a named GORM session factory that can be closed.
type SessionClient interface {
	Session(ctx context.Context) *gorm.DB
	Close() error
	SetName(name string)
	Name() string
}

// Client is a GORM database handle with optional slow-query logging, tracing,
// and Prometheus stats collection. Construct one with NewMySQLClient,
// NewClickhouseClient, or NewGormSQLiteClient.
type Client struct {
	name           string
	database       *gorm.DB
	slowThreshold  time.Duration
	tracer         trace.Tracer
	tracerInitial  sync.Once
	options        DBOptions
	statsCollector string
}

// Name returns the client name used in logs and Prometheus labels.
func (c *Client) Name() string {
	return c.name
}

// SetName sets the client name used in logs and Prometheus labels.
func (c *Client) SetName(name string) {
	c.name = name
}

// Close closes the underlying sql.DB and unregisters Prometheus stats.
func (c *Client) Close() error {
	if c == nil {
		return nil
	}
	if len(c.statsCollector) != 0 {
		collector.Unregister(c.statsCollector)
	}
	logger := logs.GetDefaultLogger()
	if sqlDB, err := c.database.DB(); err == nil {
		if err = sqlDB.Close(); err != nil {
			level.Warn(logger).Log("msg", fmt.Errorf("failed to close connect: [%s]", c.options.String()), "err", err)
		}
		level.Debug(logger).Log("msg", "MySQL connect closed")
		return err
	} else {
		level.Warn(logger).Log("msg", fmt.Errorf("failed to close connect: [%s]", c.options.String()), "err", err)
	}
	return nil
}

// Processor looks up and replaces named GORM callback processors.
type Processor interface {
	Get(name string) func(*gorm.DB)
	Replace(name string, handler func(*gorm.DB)) error
}

const instrumentationName = "github.com/sven-victor/ez-utils/clients/gorm"

// Session returns a GORM session bound to ctx with logging and tracing.
// If ctx was created with WithConnContext, that connection is reused.
func (c *Client) Session(ctx context.Context) *gorm.DB {
	host, port := c.options.GetPeer()
	c.tracerInitial.Do(func() {
		c.tracer = otel.GetTracerProvider().Tracer(instrumentationName, trace.WithInstrumentationAttributes(
			attribute.String("db.connection_string", c.options.GetConnectionString()),
			attribute.String("db.name", c.options.GetDBName()),
			attribute.String("db.system", c.options.GetType()),
			attribute.String("db.user", c.options.GetUsername()),
			attribute.String("net.peer.name", host),
			attribute.Int("net.peer.port", port),
		))
	})
	logger := logs.GetContextLogger(ctx)
	session := &gorm.Session{Logger: NewLogAdapter(logger, c.slowThreshold, c.tracer)}
	if conn := ctx.Value(gormConn{}); conn != nil {
		switch db := conn.(type) {
		case *gorm.DB:
			return db.Session(session)
		default:
			level.Warn(logger).Log("msg", "Unknown context value type.", "name", fmt.Sprintf("%T", gormConn{}), "value", fmt.Sprintf("%T", conn))
		}
	}
	return c.database.Session(session).WithContext(ctx)
}

// ConnType is the type constraint for connections stored by WithConnContext.
type ConnType interface {
	*gorm.DB
}

// WithConnContext stores conn in ctx so later Session calls reuse it,
// for example to keep work on a single transaction.
func WithConnContext[T ConnType](ctx context.Context, conn T) context.Context {
	return context.WithValue(ctx, gormConn{}, conn)
}

type gormConn struct{}
