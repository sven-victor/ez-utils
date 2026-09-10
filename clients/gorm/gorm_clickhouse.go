package gorm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/go-kit/log/level"
	"github.com/prometheus/common/model"
	clickhousedriver "gorm.io/driver/clickhouse"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"

	logs "github.com/sven-victor/ez-utils/log"
	"github.com/sven-victor/ez-utils/safe"
	"github.com/sven-victor/ez-utils/signals"
	w "github.com/sven-victor/ez-utils/wrapper"
)

// ClickhouseOptions configures a ClickHouse GORM client. Use NewClickhouseOptions
// for defaults, then pass the value to NewClickhouseClient.
type ClickhouseOptions struct {
	Host     string      `json:"host,omitempty" yaml:"host" mapstructure:"host"`
	Username string      `json:"username,omitempty" yaml:"username" mapstructure:"username"`
	Password safe.String `json:"password,omitempty" yaml:"password" mapstructure:"password"`
	// Schema is the database name.
	Schema             string `json:"schema,omitempty" yaml:"schema" mapstructure:"schema"`
	MaxIdleConnections int32  `json:"max_idle_connections,omitempty" yaml:"max_idle_connections" mapstructure:"max_idle_connections"`
	MaxOpenConnections int32  `json:"max_open_connections,omitempty" yaml:"max_open_connections" mapstructure:"max_open_connections"`
	// MaxConnectionLifeTime is the maximum lifetime of a pooled connection.
	MaxConnectionLifeTime *model.Duration `json:"max_connection_life_time,omitempty" yaml:"max_connection_life_time" mapstructure:"max_connection_life_time"`
	// TablePrefix is prepended to GORM table names. A non-empty prefix also
	// enables singular table names.
	TablePrefix string `json:"table_prefix,omitempty" yaml:"table_prefix" mapstructure:"table_prefix"`
	// SlowThreshold logs queries slower than this duration at warn level.
	SlowThreshold *model.Duration `json:"slow_threshold,omitempty" yaml:"slow_threshold" mapstructure:"slow_threshold"`
	DialTimeout   *model.Duration `json:"dial_timeout,omitempty" yaml:"dial_timeout" mapstructure:"dial_timeout"`
	ReadTimeout   *model.Duration `json:"read_timeout,omitempty" yaml:"read_timeout" mapstructure:"read_timeout"`
	// MaxExecutionTime is the ClickHouse max_execution_time query setting.
	MaxExecutionTime *model.Duration `json:"max_execution_time,omitempty" yaml:"max_execution_time" mapstructure:"max_execution_time"`
}

// Equal reports whether x matches options.
func (x *ClickhouseOptions) Equal(options ClickhouseOptions) bool {
	return !(x.Host != options.Host ||
		x.Username != options.Username ||
		!x.Password.Equal(options.Password) ||
		x.Schema != options.Schema ||
		x.MaxIdleConnections != options.MaxIdleConnections ||
		x.MaxOpenConnections != options.MaxOpenConnections ||
		!durationEqual(x.MaxConnectionLifeTime, options.MaxConnectionLifeTime) ||
		x.TablePrefix != options.TablePrefix ||
		!durationEqual(x.SlowThreshold, options.SlowThreshold) ||
		!durationEqual(x.DialTimeout, options.DialTimeout) ||
		!durationEqual(x.ReadTimeout, options.ReadTimeout) ||
		!durationEqual(x.MaxExecutionTime, options.MaxExecutionTime))
}

// GetPeer returns the host and port, defaulting the port to 9000.
func (x *ClickhouseOptions) GetPeer() (string, int) {
	host, port, found := strings.Cut(x.Host, ":")
	if found && len(port) > 0 {
		portNum, err := strconv.Atoi(port)
		if err == nil {
			return host, portNum
		}
	}
	return x.Host, 9000
}

// String returns a sanitized clickhouse://user@host/schema URL without the password.
func (x *ClickhouseOptions) String() string {
	return fmt.Sprintf("%s://%s@%s/%s", x.GetType(), x.Username, x.Host, x.Schema)
}

// GetConnectionString returns a clickhouse://host connection string for tracing.
func (x *ClickhouseOptions) GetConnectionString() string {
	return fmt.Sprintf("clickhouse://%s", x.Host)
}

// GetDBName returns the ClickHouse database name.
func (x *ClickhouseOptions) GetDBName() string {
	return x.Schema
}

// GetUsername returns the ClickHouse username.
func (x *ClickhouseOptions) GetUsername() string {
	return x.Username
}

// GetType returns "clickhouse".
func (x *ClickhouseOptions) GetType() string {
	return "clickhouse"
}

func openClickhouseConn(ctx context.Context, slowThreshold time.Duration, options *ClickhouseOptions, autoCreateSchema bool) (*gorm.DB, error) {
	logger := logs.GetContextLogger(ctx)
	dsn, err := options.GetDSN()
	if err != nil {
		return nil, err
	}

	db, err := gorm.Open(
		clickhousedriver.New(clickhousedriver.Config{
			DSN:                dsn,
			DefaultCompression: "LZ4",
		}), &gorm.Config{
			NamingStrategy: schema.NamingStrategy{
				TablePrefix:   options.TablePrefix,
				SingularTable: options.TablePrefix != "",
			},
			Logger:                                   NewLogAdapter(logger, slowThreshold, nil),
			DisableForeignKeyConstraintWhenMigrating: true,
		},
	)
	if err != nil && autoCreateSchema {
		if clickhouseErr, ok := err.(*clickhouse.Exception); ok {
			if clickhouseErr.Code == 1049 {
				level.Info(logger).Log("msg", fmt.Sprintf("auto create schema: %s", options.Schema))
				tmpOpts := *options
				tmpOpts.Schema = "clickhouse"
				db, err = openClickhouseConn(ctx, slowThreshold, &tmpOpts, false)
				if err != nil {
					return nil, err
				}
				err = db.Exec(fmt.Sprintf("CREATE DATABASE `%s`", options.Schema)).Error
				if err != nil {
					return nil, err
				}
				if sqlDB, err := db.DB(); err == nil {
					defer sqlDB.Close()
				}

				return openClickhouseConn(ctx, slowThreshold, options, false)
			}
		}
	}
	return db, err
}

// NewClickhouseClient opens a ClickHouse connection and returns a Client named name.
// If the database does not exist, it is created. The process signal handler
// closes the connection on shutdown, and Prometheus stats are registered.
func NewClickhouseClient(ctx context.Context, name string, options ClickhouseOptions) (clt *Client, err error) {
	clt = new(Client)
	clt.options = &options
	logger := logs.GetContextLogger(ctx)
	if options.SlowThreshold != nil {
		clt.slowThreshold = time.Duration(*options.SlowThreshold)
		if err != nil {
			level.Error(logger).Log("msg", fmt.Errorf("failed to connect to clickhouse server: [%s@%s]", options.Username, options.Host), "err", fmt.Errorf("`slow_threshold` option is invalid: %s", err))
			return nil, err
		}
	}
	clt.name = name
	level.Debug(logger).Log("msg", "connect to clickhouse server",
		"host", options.Host, "username", options.Username,
		"schema", options.Schema)

	db, err := openClickhouseConn(ctx, clt.slowThreshold, &options, true)
	if err != nil {
		level.Error(logger).Log("msg", fmt.Errorf("failed to connect to clickhouse server: [%s@%s]", options.Username, options.Host), "err", err)
		return nil, err
	}

	{
		sqlDB, err := db.DB()
		if err != nil {
			level.Error(logger).Log("msg", fmt.Errorf("failed to connect to clickhouse server: [%s@%s]", options.Username, options.Host), "err", err)
			return nil, err
		}
		sqlDB.SetMaxIdleConns(int(options.MaxIdleConnections))
		sqlDB.SetConnMaxLifetime(options.GetStdMaxConnectionLifeTime())
		sqlDB.SetMaxOpenConns(int(options.MaxOpenConnections))
	}

	stopCh := signals.SetupSignalHandler(logger)
	stopCh.PreStop(signals.LevelDB, func() {
		if sqlDB, err := db.DB(); err == nil {
			if err = sqlDB.Close(); err != nil {
				level.Warn(logger).Log("msg", fmt.Errorf("failed to close clickhouse connect: [%s@%s]", options.Username, options.Host), "err", err)
			}
		} else {
			level.Warn(logger).Log("msg", fmt.Errorf("failed to close clickhouse connect: [%s@%s]", options.Username, options.Host), "err", err)
		}
		level.Debug(logger).Log("msg", "Clickhouse connect closed")
	})

	level.Info(logger).Log("msg", "connected to clickhouse server",
		"host", options.Host, "username", options.Username,
		"schema", options.Schema)
	clt.database = db
	clt.statsCollector = collector.Register(clt)
	return clt, nil
}

// GetStdMaxConnectionLifeTime returns MaxConnectionLifeTime, or 30s if unset.
func (x *ClickhouseOptions) GetStdMaxConnectionLifeTime() time.Duration {
	if x != nil && x.MaxConnectionLifeTime != nil {
		return time.Duration(*x.MaxConnectionLifeTime)
	}
	return time.Second * 30
}

// GetDSN builds a clickhouse:// DSN including timeouts when they are set.
func (x *ClickhouseOptions) GetDSN() (string, error) {
	var u url.URL
	u.Host = x.Host
	passwd, err := x.Password.UnsafeString()
	if err != nil {
		return "", err
	}
	u.User = url.UserPassword(x.Username, passwd)
	u.Scheme = "clickhouse"
	u.Path = fmt.Sprintf("/%s", x.Schema)
	var q url.Values
	if x.DialTimeout != nil {
		q.Set("dial_timeout", x.DialTimeout.String())
	}
	if x.ReadTimeout != nil {
		q.Set("read_timeout", x.ReadTimeout.String())
	}
	if x.MaxExecutionTime != nil {
		q.Set("max_execution_time", x.MaxExecutionTime.String())
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

// NewClickhouseOptions returns ClickhouseOptions with local-development defaults
// (table prefix t_ and an idas database).
func NewClickhouseOptions() *ClickhouseOptions {
	return &ClickhouseOptions{
		MaxIdleConnections:    2,
		MaxOpenConnections:    100,
		MaxConnectionLifeTime: (*model.Duration)(w.P(30 * time.Second)),
		TablePrefix:           "t_",
		Host:                  "localhost",
		Schema:                "idas",
		Username:              "idas",
	}
}

// ClickhouseClient is a Client that keeps its ClickhouseOptions for JSON round-trips.
type ClickhouseClient struct {
	*Client
	options *ClickhouseOptions
}

// Options returns a copy of the ClickHouse options used to open the client.
func (c ClickhouseClient) Options() ClickhouseOptions {
	return *c.options
}

// SetOptions replaces the stored ClickHouse options without reconnecting.
func (c *ClickhouseClient) SetOptions(o *ClickhouseOptions) {
	c.options = o
}

// MarshalJSON encodes the ClickHouse options.
func (c ClickhouseClient) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.options)
}

// UnmarshalJSON decodes ClickhouseOptions and opens a new connection.
func (c *ClickhouseClient) UnmarshalJSON(data []byte) (err error) {
	if c.options == nil {
		c.options = NewClickhouseOptions()
	}
	if err = json.Unmarshal(data, c.options); err != nil {
		return err
	}
	if c.Client, err = NewClickhouseClient(context.Background(), "", *c.options); err != nil {
		return err
	}
	return
}
