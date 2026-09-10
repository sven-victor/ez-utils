package gorm

import (
	"context"
	"database/sql/driver"
	"encoding/base64"
	"encoding/json"
	"fmt"

	gosqlite "github.com/glebarez/go-sqlite"
	"github.com/glebarez/sqlite"
	"github.com/go-kit/log/level"
	"github.com/gogo/protobuf/types"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"

	"github.com/sven-victor/ez-utils/log"
	"github.com/sven-victor/ez-utils/signals"
)

// SQLiteOptions configures a SQLite GORM client. Use NewSQLiteOptions for
// defaults, then pass the value to NewSQLiteClient or NewGormSQLiteClient.
type SQLiteOptions struct {
	Path        string `protobuf:"bytes,1,opt,name=path,proto3" json:"path,omitempty"`
	TablePrefix string `protobuf:"bytes,2,opt,name=table_prefix,json=tablePrefix,proto3" json:"table_prefix,omitempty"`
	// SlowThreshold logs queries slower than this duration at warn level.
	SlowThreshold        *types.Duration `protobuf:"bytes,12,opt,name=slow_threshold,json=slowThreshold,proto3" json:"slow_threshold,omitempty"`
	XXX_NoUnkeyedLiteral struct{}        `json:"-"` //nolint:revive
	XXX_unrecognized     []byte          `json:"-"` //nolint:revive
	XXX_sizecache        int32           `json:"-"` //nolint:revive
}

// String returns the sqlite://path connection string.
func (o *SQLiteOptions) String() string {
	return o.GetConnectionString()
}

// GetPeer returns the database file path and port 0.
func (o SQLiteOptions) GetPeer() (string, int) {
	return o.Path, 0
}

// GetConnectionString returns a sqlite://path connection string for tracing.
func (o SQLiteOptions) GetConnectionString() string {
	return fmt.Sprintf("sqlite://%s", o.Path)
}

// GetDBName returns "main".
func (o SQLiteOptions) GetDBName() string {
	return "main"
}

// GetUsername returns an empty string; SQLite has no username.
func (o SQLiteOptions) GetUsername() string {
	return ""
}

// GetType returns "sqlite".
func (o SQLiteOptions) GetType() string {
	return "sqlite"
}

func init() {
	gosqlite.MustRegisterDeterministicScalarFunction("from_base64", 1, func(ctx *gosqlite.FunctionContext, args []driver.Value) (driver.Value, error) {
		switch argTyped := args[0].(type) {
		case string:
			return base64.StdEncoding.DecodeString(argTyped)
		default:
			return nil, fmt.Errorf("unsupported type: %T", args[0])
		}
	})
}

// NewSQLiteClient opens a SQLite database and returns an SQLiteClient named name.
func NewSQLiteClient(ctx context.Context, name string, options *SQLiteOptions) (clt *SQLiteClient, err error) {
	client, err := NewGormSQLiteClient(ctx, name, options)
	if err != nil {
		return nil, err
	}
	return &SQLiteClient{Client: client, options: options}, nil
}

// NewGormSQLiteClient opens a SQLite database and returns a Client named name.
// The process signal handler closes the connection on shutdown, and Prometheus
// stats are registered.
func NewGormSQLiteClient(ctx context.Context, name string, options *SQLiteOptions) (clt *Client, err error) {
	clt = new(Client)
	clt.options = options
	logger := log.GetContextLogger(ctx)
	if options.SlowThreshold != nil {
		clt.slowThreshold, err = types.DurationFromProto(options.SlowThreshold)
		if err != nil {
			level.Error(logger).Log("msg", fmt.Sprintf("failed to connect to SQLite database: %s", options.Path), "err", fmt.Errorf("`slow_threshold` option is invalid: %s", err))
			return nil, err
		}
	}
	clt.name = name

	level.Debug(logger).Log("msg", "connect to sqlite", "dsn", options.Path)
	db, err := gorm.Open(sqlite.Open(options.Path), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			TablePrefix:   "t_",
			SingularTable: true,
		},
		Logger:                                   NewLogAdapter(logger, clt.slowThreshold, nil),
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		level.Error(logger).Log("msg", fmt.Sprintf("failed to connect to SQLite database: %s", options.Path), "err", err)
		return nil, fmt.Errorf("failed to connect to SQLite database: %s: %s", options.Path, err)
	}
	stopCh := signals.SetupSignalHandler(logger)
	stopCh.PreStop(signals.LevelDB, func() {
		if sqlDB, err := db.DB(); err == nil {
			if err = sqlDB.Close(); err != nil {
				level.Warn(logger).Log("msg", "Failed to close SQLite database", "err", err)
			}
		}
		level.Debug(logger).Log("msg", "Sqlite connect closed")
	})
	clt.database = db
	clt.statsCollector = collector.Register(clt)
	return clt, nil
}

// NewSQLiteOptions returns SQLiteOptions pointing at data.db with table prefix t_.
func NewSQLiteOptions() *SQLiteOptions {
	return &SQLiteOptions{
		Path:        "data.db",
		TablePrefix: "t_",
	}
}

// SQLiteClient is a Client that keeps its SQLiteOptions for JSON round-trips.
type SQLiteClient struct {
	*Client
	options *SQLiteOptions
}

// MarshalJSON encodes the SQLite options.
func (c SQLiteClient) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.options)
}

// UnmarshalJSON decodes SQLiteOptions and opens a new connection.
func (c *SQLiteClient) UnmarshalJSON(data []byte) (err error) {
	if c.options == nil {
		c.options = NewSQLiteOptions()
	}

	if err = json.Unmarshal(data, c.options); err != nil {
		return err
	}
	if c.Client, err = NewGormSQLiteClient(context.Background(), "", c.options); err != nil {
		return err
	}
	return
}

// Close closes the underlying sql.DB.
func (c SQLiteClient) Close() error {
	if sqlDB, err := c.database.DB(); err == nil {
		return sqlDB.Close()
	} else {
		return err
	}
}
