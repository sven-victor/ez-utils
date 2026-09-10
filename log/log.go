// Copyright 2026 Sven Victor
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package log provides leveled, contextual loggers built on go-kit/log.
package log

import (
	"context"
	"fmt"
	"io"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	rotatelogs "github.com/lestrrat-go/file-rotatelogs"

	"github.com/sven-victor/ez-utils/capacity"
	errors "github.com/sven-victor/ez-utils/errors"
	g "github.com/sven-victor/ez-utils/generator"
	w "github.com/sven-victor/ez-utils/wrapper"
)

// TimestampFormat This timestamp format differs from RFC3339Nano by using .000 instead
// of .999999999 which changes the timestamp from 9 variable to 3 fixed
// decimals (.130 instead of .130987456).
var TimestampFormat = log.TimestampFormat(
	func() time.Time { return time.Now().UTC() },
	"2006-01-02T15:04:05.000Z07:00",
)

type callerName struct{}

func (callerName) String() string {
	return "caller"
}

// CallerName is the log key used for the call-site file and line number.
var CallerName = callerName{}

const (
	// TraceIdName is the log and context key used to store a trace identifier.
	TraceIdName = "traceId"
)

// DefaultCaller is a Valuer that records the call site five frames above the logger.
var DefaultCaller = Caller(5)

var sourceDir = GetSourceCodeDir("log/log.go")

// GetSourceCodeDir returns the directory prefix of the current source file after
// trimming codeRelativePath. Optional depth selects a different runtime.Caller frame.
func GetSourceCodeDir(codeRelativePath string, depth ...int) string {
	var currentFile string
	if len(depth) == 0 {
		_, currentFile, _, _ = runtime.Caller(1)
	} else {
		_, currentFile, _, _ = runtime.Caller(depth[0])
	}
	return strings.TrimSuffix(currentFile, codeRelativePath)
}

// SetSourceCodeDir overrides the prefix stripped from caller file paths.
func SetSourceCodeDir(d string) {
	sourceDir = d
}

var callerFormatter = func(file string, line int) string {
	return strings.TrimPrefix(file, sourceDir) + ":" + strconv.Itoa(line)
}

// SetCallerFormatter replaces the function that formats caller file and line values.
func SetCallerFormatter(f func(file string, line int) string) {
	callerFormatter = f
}

// Caller returns a Valuer that records the file and line at the given call depth.
func Caller(depth int) log.Valuer {
	return func() interface{} {
		_, file, line, _ := runtime.Caller(depth)
		return callerFormatter(file, line)
	}
}

func newWriterFromConfig(c Config) io.Writer {
	var writer io.Writer
	if len(c.FilePath) != 0 {
		switch c.FilePath {
		case "/dev/stdin":
			writer = os.Stdin
		case "/dev/stdout":
			writer = os.Stdout
		case "/dev/stderr":
			writer = os.Stderr
		default:
			stat, err := os.Stat(c.FilePath)
			if err == nil && stat.Mode()&os.ModeType != 0 {
				_, _ = fmt.Fprintf(os.Stderr, "unknown file mode: %s", stat.Mode())
				//nolint:gofumpt
				if writer, err = os.OpenFile(c.FilePath, os.O_APPEND|os.O_RDWR, 0600); err != nil {
					_, _ = fmt.Fprintf(os.Stderr, "[WARN]failed to open file %s, use /dev/stdout (console stdout)", err)
					return os.Stdout
				}
			} else if (err != nil && os.IsNotExist(err)) || err == nil {
				fileRotationSize := capacity.Capacities(0)
				if c.FileRotationSize != "" && c.FileRotationSize != "0" {
					fileRotationSize, err = capacity.ParseCapacities(c.FileRotationSize)
					if err != nil {
						_, _ = fmt.Fprintf(os.Stderr, "[WARN] Failed to parse parameter file-rotation-size, do not rotate logs by size: %s", err)
						fileRotationSize = 0
					}
				}
				if writer, err = rotatelogs.New(
					c.FilePath+"-%Y%m%d%H%M",
					rotatelogs.WithLinkName(c.FilePath),
					rotatelogs.WithMaxAge(c.FileMaxAge),
					rotatelogs.WithRotationSize(int64(fileRotationSize)),
					rotatelogs.WithRotationTime(c.FileRotationTime),
				); err != nil {
					_, _ = fmt.Fprintf(os.Stderr, "[WARN]failed to create log rotate %s, use /dev/stdout (console stdout)", err)
					return os.Stdout
				}
			} else {
				_, _ = fmt.Fprintf(os.Stderr, "[WARN]failed to open log file, unknown err: %s, ", err)
				return os.Stdout
			}
		}
	} else {
		writer = os.Stdout
	}
	return writer
}

// New returns a leveled go-kit logger annotated with a timestamp and caller.
// Output goes to Config.FilePath (or stdout when unset), unless WithWriter is given.
func New(opts ...NewLoggerOption) log.Logger {
	o := newLoggerOptions{}
	for _, opt := range opts {
		opt(&o)
	}
	config := o.config
	var l log.Logger
	if config == nil {
		config = DefaultLoggerConfig
	}
	if config.Format == nil {
		config.Format = w.P[AllowedFormat](FormatLogfmt)
	}
	if config.Level == nil {
		config.Level = w.P[AllowedLevel](LevelInfo)
	}
	if o.w == nil {
		o.w = log.NewSyncWriter(newWriterFromConfig(*config))
	}

	if val, ok := registeredLogFormat.Load(*config.Format); ok {
		if createFunc, ok := val.(LoggerCreateFunc); ok {
			l = createFunc(o.w)
		} else {
			l = log.NewLogfmtLogger(o.w)
			_ = level.Warn(l).Log("msg", fmt.Errorf("log output format %s is not registered, using logfmt", *config.Format))
		}
	} else {
		l = log.NewLogfmtLogger(o.w)
		_ = level.Warn(l).Log("msg", fmt.Errorf("log output format %s is not registered, using logfmt", *config.Format))
	}
	l = level.NewFilter(log.With(&extLogger{logger: l, w: o.w}, "ts", TimestampFormat, CallerName, DefaultCaller), config.Level.getOption())
	return l
}

var (
	rootLoggerOnce sync.Once
	rootLogger     log.Logger
)

// SetDefaultLogger replaces the process-wide default logger.
func SetDefaultLogger(logger log.Logger) {
	rootLogger = logger
}

// GetDefaultLogger returns the process-wide default logger, creating one on first use.
func GetDefaultLogger() log.Logger {
	rootLoggerOnce.Do(func() {
		if rootLogger == nil {
			rootLogger = New()
		}
	})
	return rootLogger
}

// NewLoggerOption configures a logger created by New, NewTraceLogger, or NewContextLogger.
type NewLoggerOption func(*newLoggerOptions)

// WithTraceId sets the trace identifier attached to a trace logger.
func WithTraceId(traceId string) NewLoggerOption {
	return func(o *newLoggerOptions) {
		o.traceId = traceId
	}
}

// WithLogger uses l as the base logger instead of the default logger.
func WithLogger(l log.Logger) NewLoggerOption {
	return func(o *newLoggerOptions) {
		o.l = l
	}
}

// WithWriter writes log output to w instead of the writer derived from Config.
func WithWriter(w io.Writer) NewLoggerOption {
	return func(o *newLoggerOptions) {
		o.w = w
	}
}

// WithConfig applies c when creating a logger.
func WithConfig(c *Config) NewLoggerOption {
	return func(o *newLoggerOptions) {
		o.config = c
	}
}

// WithKeyValues attaches additional key/value pairs to the logger.
func WithKeyValues(keyvals ...interface{}) NewLoggerOption {
	return func(o *newLoggerOptions) {
		o.kvs = keyvals
	}
}

type newLoggerOptions struct {
	traceId string
	l       log.Logger
	w       io.Writer
	kvs     []interface{}
	config  *Config
}

// NewTraceLogger returns a logger annotated with a trace identifier.
func NewTraceLogger(options ...NewLoggerOption) log.Logger {
	o := newLoggerOptions{traceId: NewTraceId(), l: rootLogger}
	for _, f := range options {
		f(&o)
	}
	if o.l == nil {
		return log.With(GetDefaultLogger(), TraceIdName, o.traceId)
	}
	if len(o.kvs) != 0 {
		o.l = log.With(o.l, o.kvs...)
		o.kvs = nil
	}
	return log.With(o.l, TraceIdName, o.traceId)
}

// NewTraceId returns a new unique trace identifier.
func NewTraceId() string {
	return g.NewId("logging")
}

// GetTraceId returns the trace identifier stored in ctx, or a new one if none is present.
func GetTraceId(ctx context.Context) string {
	s, _ := ctx.Value(contextTraceId{}).(string)
	if len(s) == 0 {
		return NewTraceId()
	}
	return s
}

type contextTraceId struct{}

// NewContextLogger stores a trace identifier and logger in a child of parent and returns both.
func NewContextLogger(parent context.Context, opts ...NewLoggerOption) (context.Context, log.Logger) {
	o := newLoggerOptions{}
	for _, opt := range opts {
		opt(&o)
	}
	if len(o.traceId) == 0 {
		o.traceId = NewTraceId()
	}
	if o.l == nil {
		o.l = NewTraceLogger(WithTraceId(o.traceId))
	}
	if len(o.kvs) != 0 {
		o.l = log.With(o.l, o.kvs...)
		o.kvs = nil
	}
	return context.WithValue(context.WithValue(parent, contextTraceId{}, o.traceId), contextLogger{}, o.l), o.l
}

type contextLogger struct{}

// GetContextLogger returns the logger stored in ctx, or a new trace logger if none is present.
func GetContextLogger(ctx context.Context, options ...Option) log.Logger {
	l, ok := ctx.Value(contextLogger{}).(log.Logger)
	if !ok {
		l = NewTraceLogger()
	}
	for _, option := range options {
		l = option(l)
	}
	return l
}

// Option wraps a logger to attach extra fields when retrieving a context logger.
type Option func(l log.Logger) log.Logger

// WithCaller returns an Option that records the call site at the given stack layer.
func WithCaller(layer int) Option {
	return func(l log.Logger) log.Logger {
		return log.With(l, CallerName, Caller(layer))
	}
}

// WithMethod returns an Option that records the calling function name as a "method" field.
func WithMethod(skip ...int) Option {
	pc := make([]uintptr, 1)
	if len(skip) > 0 {
		runtime.Callers(skip[0], pc)
	} else {
		runtime.Callers(2, pc)
	}
	funcName := strings.SplitAfterN(runtime.FuncForPC(pc[0]).Name(), ".", 2)
	return func(l log.Logger) log.Logger {
		return log.With(l, "method", funcName[len(funcName)-1])
	}
}

// KeyName is a typed log key that implements fmt.Stringer.
type KeyName string

// String returns the key name as a string.
func (n KeyName) String() string {
	return string(n)
}

// WrapKeyName returns name as a KeyName so it can be used as a log key.
func WrapKeyName(name string) fmt.Stringer {
	return KeyName(name)
}

type nopLogger struct{}

func (n *nopLogger) Log(keyvals ...interface{}) error {
	return nil
}

// NewNopLogger returns a logger that discards all log records.
func NewNopLogger() log.Logger {
	return &nopLogger{}
}

type teeLogger []log.Logger

func (t teeLogger) Log(keyvals ...interface{}) error {
	errs := errors.NewErrors(500, "")
	for _, l := range t {
		if err := l.Log(keyvals...); err != nil {
			errs.Append(err)
		}
	}
	if errs.HasError() {
		return errs
	}
	return nil
}

// NewTeeLogger returns a logger that writes each record to every logger in lgs.
func NewTeeLogger(lgs ...log.Logger) log.Logger {
	return teeLogger(lgs)
}
