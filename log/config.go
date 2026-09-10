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

package log

import (
	"io"
	"sync"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/pkg/errors"
)

// AllowedLevel is a settable identifier for the minimum level a log entry
// must have.
type AllowedLevel string

func (l AllowedLevel) getOption() level.Option {
	switch l {
	case LevelDebug:
		return level.AllowDebug()
	case LevelInfo:
		return level.AllowInfo()
	case LevelWarn:
		return level.AllowWarn()
	case LevelError:
		return level.AllowError()
	default:
		return level.AllowWarn()
	}
}

// UnmarshalYAML implements yaml.Unmarshaler by parsing a log level string.
func (l *AllowedLevel) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	type plain string
	if err := unmarshal((*plain)(&s)); err != nil {
		return err
	}
	return l.Set(s)
}

// String returns the log level name.
func (l AllowedLevel) String() string {
	return string(l)
}

// Valid reports whether l is a recognized log level.
func (l AllowedLevel) Valid() error {
	switch l {
	case LevelDebug, LevelInfo, LevelWarn, LevelError:
		return nil
	default:
		return errors.Errorf(`unrecognized log level "%s"`, l)
	}
}

// Set updates the value of the allowed level.
func (l *AllowedLevel) Set(s string) error {
	lvl := AllowedLevel(s)
	if len(s) == 0 {
		s = string(LevelWarn)
	}
	if err := lvl.Valid(); err != nil {
		return err
	}
	*l = AllowedLevel(s)
	return nil
}

const (
	// LevelDebug allows debug, info, warn, and error log entries.
	LevelDebug AllowedLevel = "debug"
	// LevelInfo allows info, warn, and error log entries.
	LevelInfo AllowedLevel = "info"
	// LevelWarn allows warn and error log entries.
	LevelWarn AllowedLevel = "warn"
	// LevelError allows only error log entries.
	LevelError AllowedLevel = "error"
)

// LoggerCreateFunc builds a logger that writes to w.
type LoggerCreateFunc func(w io.Writer) log.Logger

var registeredLogFormat sync.Map

// RegisterLogFormat registers f as the constructor for logFmt.
func RegisterLogFormat(logFmt AllowedFormat, f LoggerCreateFunc) {
	registeredLogFormat.Store(logFmt, f)
}

// GetRegisteredLogFormats returns the names of all registered log formats.
func GetRegisteredLogFormats() []AllowedFormat {
	var fmts []AllowedFormat
	registeredLogFormat.Range(func(key, _ any) bool {
		fmts = append(fmts, key.(AllowedFormat))
		return true
	})
	return fmts
}

// AllowedFormat is a settable identifier for the output format that the logger can have.
type AllowedFormat string

// String returns the format name.
func (f AllowedFormat) String() string {
	return string(f)
}

// Valid reports whether f is a registered log format.
func (f AllowedFormat) Valid() error {
	_, ok := registeredLogFormat.Load(f)
	if !ok {
		return errors.Errorf("unrecognized log format %s", f)
	}
	return nil
}

// Set updates the value of the allowed format.
func (f *AllowedFormat) Set(s string) error {
	format := AllowedFormat(s)
	if err := format.Valid(); err != nil {
		return err
	}
	*f = format
	return nil
}

const (
	// FormatJSON encodes each log record as JSON.
	FormatJSON AllowedFormat = "json"
	// FormatLogfmt encodes each log record in logfmt.
	FormatLogfmt AllowedFormat = "logfmt"
)

// Config is a struct containing configurable settings for the logger
type Config struct {
	// Level is the minimum severity that will be logged.
	Level *AllowedLevel
	// Format selects the encoder used for log records.
	Format *AllowedFormat
	// FilePath is the log file path, or a special device such as /dev/stdout.
	FilePath string
	// FileMaxAge is how long rotated log files are retained.
	FileMaxAge time.Duration
	// FileRotationSize is the maximum size of a log file before rotation, as a capacity string.
	FileRotationSize string
	// FileRotationTime is how often log files are rotated by time.
	FileRotationTime time.Duration
}

// MustNewConfig returns a Config with the given level and format, panicking on invalid values.
func MustNewConfig(level string, format string) *Config {
	cfg := &Config{Level: new(AllowedLevel), Format: new(AllowedFormat)}
	if err := cfg.Level.Set(level); err != nil {
		panic(err)
	}
	if err := cfg.Format.Set(format); err != nil {
		panic(err)
	}
	return cfg
}

// DefaultLoggerConfig is the process-wide default Config used when none is provided.
var DefaultLoggerConfig *Config

func init() {
	RegisterLogFormat(FormatLogfmt, log.NewLogfmtLogger)
	RegisterLogFormat(FormatJSON, log.NewJSONLogger)
	DefaultLoggerConfig = MustNewConfig("info", "logfmt")
}
