package gorm

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/go-kit/log/level"
	"github.com/go-sql-driver/mysql"
	"github.com/prometheus/common/model"
	mysqldriver "gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"

	"github.com/sven-victor/ez-utils/clients/tls"
	g "github.com/sven-victor/ez-utils/generator"
	logs "github.com/sven-victor/ez-utils/log"
	"github.com/sven-victor/ez-utils/safe"
	"github.com/sven-victor/ez-utils/signals"
	w "github.com/sven-victor/ez-utils/wrapper"
)

// TLSOptions is a MySQL TLS setting that unmarshals from either a registered
// config name or a full tls.TLSOptions object. When a full object is provided,
// UnmarshalJSON registers it with the MySQL driver under a generated name.
type TLSOptions struct {
	options *tls.TLSOptions
	name    string
}

// MarshalJSON encodes the nested TLS options, or the registered config name
// when no options object is set.
func (o TLSOptions) MarshalJSON() ([]byte, error) {
	if o.options == nil {
		return []byte(o.name), nil
	}
	return json.Marshal(o.options)
}

// UnmarshalJSON accepts a JSON string naming a registered MySQL TLS config,
// or a tls.TLSOptions object which is registered with the MySQL driver.
func (o *TLSOptions) UnmarshalJSON(data []byte) (err error) {
	if err = json.Unmarshal(data, &o.name); err == nil {
		return nil
	}
	if o.options == nil {
		o.options = &tls.TLSOptions{}
	}
	if err = json.Unmarshal(data, o.options); err != nil {
		return err
	}
	o.name = g.NewId()
	tlsConfig, err := tls.NewTLSConfig(o.options)
	if err != nil {
		return err
	}
	return mysql.RegisterTLSConfig(o.name, tlsConfig)
}

// Equal reports whether o and o2 describe the same TLS options.
func (o *TLSOptions) Equal(o2 *TLSOptions) bool {
	if (o == nil && o2 == nil) || (o.options == nil && o2.options == nil) {
		return true
	}
	return o != nil && o2 != nil && reflect.DeepEqual(o.options, o2.options)
}

// MySQLOptions configures a MySQL GORM client. Use NewMySQLOptions for
// defaults, then pass the value to NewMySQLClient.
type MySQLOptions struct {
	Host     string      `json:"host,omitempty"`
	Username string      `json:"username,omitempty"`
	Password safe.String `json:"password,omitempty"`
	// Schema is the database (schema) name.
	Schema             string `json:"schema,omitempty"`
	MaxIdleConnections int32  `json:"max_idle_connections,omitempty" yaml:"max_idle_connections" mapstructure:"max_idle_connections"`
	MaxOpenConnections int32  `json:"max_open_connections,omitempty" yaml:"max_open_connections" mapstructure:"max_open_connections"`
	// MaxConnectionLifeTime is the maximum lifetime of a pooled connection.
	MaxConnectionLifeTime *model.Duration `json:"max_connection_life_time,omitempty" yaml:"max_connection_life_time" mapstructure:"max_connection_life_time"`
	Charset               string          `json:"charset,omitempty"`
	Collation             string          `json:"collation,omitempty"`
	// TablePrefix is prepended to GORM table names. A non-empty prefix also
	// enables singular table names.
	TablePrefix string `json:"table_prefix,omitempty" yaml:"table_prefix" mapstructure:"table_prefix"`
	// SlowThreshold logs queries slower than this duration at warn level.
	SlowThreshold *model.Duration `json:"slow_threshold,omitempty" yaml:"slow_threshold" mapstructure:"slow_threshold"`
	TLSConfig     *TLSOptions     `json:"tls_config" yaml:"tls_config" mapstructure:"tls_config"`
	// EnableCompression enables MySQL protocol compression.
	EnableCompression bool `json:"enable_compression,omitempty" yaml:"enable_compression" mapstructure:"enable_compression"`
}

func durationEqual(dur1, dur2 *model.Duration) bool {
	if dur1 == nil && dur2 == nil {
		return true
	}
	if dur1 != nil && dur2 != nil && dur1.String() == dur2.String() {
		return true
	}
	return false
}

// Equal reports whether x matches options.
func (x *MySQLOptions) Equal(options MySQLOptions) bool {
	return !(x.Host != options.Host ||
		x.Username != options.Username ||
		!x.Password.Equal(options.Password) ||
		x.Schema != options.Schema ||
		x.MaxIdleConnections != options.MaxIdleConnections ||
		x.MaxOpenConnections != options.MaxOpenConnections ||
		!durationEqual(x.MaxConnectionLifeTime, options.MaxConnectionLifeTime) ||
		x.Charset != options.Charset ||
		x.Collation != options.Collation ||
		x.TablePrefix != options.TablePrefix ||
		!durationEqual(x.SlowThreshold, options.SlowThreshold) ||
		!x.TLSConfig.Equal(x.TLSConfig))
}

// String returns a sanitized mysql://user@host/schema URL without the password.
func (x *MySQLOptions) String() string {
	return fmt.Sprintf("%s://%s@%s/%s", x.GetType(), x.Username, x.Host, x.Schema)
}

// GetPeer returns the host and port, defaulting the port to 3306.
func (x *MySQLOptions) GetPeer() (string, int) {
	host, port, found := strings.Cut(x.Host, ":")
	if found && len(port) > 0 {
		portNum, err := strconv.Atoi(port)
		if err == nil {
			return host, portNum
		}
	}
	return x.Host, 3306
}

// GetConnectionString returns a mysql://host connection string for tracing.
func (x *MySQLOptions) GetConnectionString() string {
	return fmt.Sprintf("mysql://%s", x.Host)
}

// GetDBName returns the MySQL schema name.
func (x *MySQLOptions) GetDBName() string {
	return x.Schema
}

// GetUsername returns the MySQL username.
func (x *MySQLOptions) GetUsername() string {
	return x.Username
}

// GetType returns "mysql".
func (x *MySQLOptions) GetType() string {
	return "mysql"
}

func openMysqlConn(ctx context.Context, slowThreshold time.Duration, options *MySQLOptions, autoCreateSchema bool) (*gorm.DB, error) {
	logger := logs.GetContextLogger(ctx)
	passwd, err := options.Password.UnsafeString()
	if err != nil {
		return nil, err
	}
	if options.Charset == "" {
		options.Charset = "utf8mb4"
	}
	if options.Collation == "" {
		options.Collation = "utf8mb4_general_ci"
	}
	if options.MaxOpenConnections == 0 {
		options.MaxOpenConnections = 100
	}

	var tlsConfigName string
	if options.TLSConfig != nil {
		tlsConfigName = options.TLSConfig.name
	}
	cfg := &mysql.Config{
		User:                 options.Username,
		Passwd:               passwd,
		Net:                  "tcp",
		Addr:                 options.Host,
		DBName:               options.Schema,
		Params:               map[string]string{"charset": options.Charset},
		Collation:            options.Collation,
		AllowNativePasswords: true,
		CheckConnLiveness:    true,
		ParseTime:            true,
		TLSConfig:            tlsConfigName,
	}
	if options.EnableCompression {
		cfg.Apply(mysql.EnableCompression(true))
	}
	db, err := gorm.Open(
		mysqldriver.New(mysqldriver.Config{
			DSNConfig: cfg,
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
		if mysqlErr, ok := err.(*mysql.MySQLError); ok {
			if mysqlErr.Number == 1049 && autoCreateSchema {
				level.Info(logger).Log("msg", fmt.Sprintf("auto create schema: %s", options.Schema))
				tmpOpts := *options
				tmpOpts.Schema = "mysql"
				db, err = openMysqlConn(ctx, slowThreshold, &tmpOpts, false)
				if err != nil {
					return nil, err
				}
				err = db.Exec(fmt.Sprintf("CREATE SCHEMA `%s` DEFAULT CHARACTER SET %s COLLATE %s", options.Schema, options.Charset, options.Collation)).Error
				if err != nil {
					return nil, err
				}
				if sqlDB, err := db.DB(); err == nil {
					defer sqlDB.Close()
				}

				return openMysqlConn(ctx, slowThreshold, options, false)
			}
		}
	}
	return db, err
}

// NewMySQLClient opens a MySQL connection and returns a Client named name.
// If the schema does not exist, it is created. The process signal handler
// closes the connection on shutdown, and Prometheus stats are registered.
func NewMySQLClient(ctx context.Context, name string, options MySQLOptions) (clt *Client, err error) {
	clt = new(Client)
	clt.options = &options
	logger := logs.GetContextLogger(ctx)
	if options.SlowThreshold != nil {
		clt.slowThreshold = time.Duration(*options.SlowThreshold)
	}
	clt.name = name
	level.Debug(logger).Log("msg", "connect to mysql server",
		"host", options.Host, "username", options.Username,
		"schema", options.Schema,
		"charset", options.Charset,
		"enableCompression", options.EnableCompression,
		"collation", options.Collation)

	db, err := openMysqlConn(ctx, clt.slowThreshold, &options, true)
	if err != nil {
		level.Error(logger).Log("msg", fmt.Errorf("failed to connect to mysql server: [%s@%s]", options.Username, options.Host), "err", err)
		return nil, err
	}

	{
		sqlDB, err := db.DB()
		if err != nil {
			level.Error(logger).Log("msg", fmt.Errorf("failed to connect to mysql server: [%s@%s]", options.Username, options.Host), "err", err)
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
				level.Warn(logger).Log("msg", fmt.Errorf("failed to close mysql connect: [%s@%s]", options.Username, options.Host), "err", err)
			}
		} else {
			level.Warn(logger).Log("msg", fmt.Errorf("failed to close mysql connect: [%s@%s]", options.Username, options.Host), "err", err)
		}
		level.Debug(logger).Log("msg", "MySQL connect closed")
	})
	level.Info(logger).Log("msg", "connected to mysql server",
		"host", options.Host, "username", options.Username,
		"schema", options.Schema,
		"charset", options.Charset,
		"collation", options.Collation)
	clt.database = db
	clt.statsCollector = collector.Register(clt)
	return clt, nil
}

// GetStdMaxConnectionLifeTime returns MaxConnectionLifeTime, or 30s if unset.
func (x *MySQLOptions) GetStdMaxConnectionLifeTime() time.Duration {
	if x != nil && x.MaxConnectionLifeTime != nil {
		return time.Duration(*x.MaxConnectionLifeTime)
	}
	return time.Second * 30
}

// NewMySQLOptions returns MySQLOptions with local-development defaults
// (utf8 charset, table prefix t_, and an idas schema).
func NewMySQLOptions() *MySQLOptions {
	return &MySQLOptions{
		Charset:               "utf8",
		Collation:             "utf8_general_ci",
		MaxIdleConnections:    2,
		MaxOpenConnections:    100,
		MaxConnectionLifeTime: (*model.Duration)(w.P(30 * time.Second)),
		TablePrefix:           "t_",
		Host:                  "localhost",
		Schema:                "idas",
		Username:              "idas",
	}
}

// MySQLClient is a Client that keeps its MySQLOptions for JSON round-trips.
type MySQLClient struct {
	*Client
	options *MySQLOptions
}

// Options returns a copy of the MySQL options used to open the client.
func (c MySQLClient) Options() MySQLOptions {
	return *c.options
}

// SetOptions replaces the stored MySQL options without reconnecting.
func (c *MySQLClient) SetOptions(o *MySQLOptions) {
	c.options = o
}

// MarshalJSON encodes the MySQL options.
func (c MySQLClient) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.options)
}

// UnmarshalJSON decodes MySQLOptions and opens a new connection.
func (c *MySQLClient) UnmarshalJSON(data []byte) (err error) {
	if c.options == nil {
		c.options = NewMySQLOptions()
	}
	if err = json.Unmarshal(data, c.options); err != nil {
		return err
	}
	if c.Client, err = NewMySQLClient(context.Background(), "", *c.options); err != nil {
		return err
	}
	return
}
