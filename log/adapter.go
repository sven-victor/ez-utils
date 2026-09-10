package log

import (
	"io"
	"strings"

	"github.com/go-kit/log"
)

// WriterAdapter is an io.Writer that forwards each write as a log record.
type WriterAdapter struct {
	l               log.Logger
	msgKey          string
	prefix          string
	joinPrefixToMsg bool
}

// Write logs p as a single message using the configured key and prefix rules.
func (a WriterAdapter) Write(p []byte) (n int, err error) {
	a.l.Log(a.msgKey, a.handleMessagePrefix(strings.TrimSuffix(string(p), "\n")))
	return len(p), nil
}

func (a WriterAdapter) handleMessagePrefix(msg string) string {
	if a.prefix == "" {
		return msg
	}

	msg = strings.TrimPrefix(msg, a.prefix)
	if a.joinPrefixToMsg {
		msg = a.prefix + msg
	}
	return msg
}

// MessageKey returns a WriterAdapterOption that sets the key used for the logged message.
func MessageKey(key string) WriterAdapterOption {
	return func(a *WriterAdapter) { a.msgKey = key }
}

// Prefix returns a WriterAdapterOption that strips prefix from each write.
// When joinPrefixToMsg is true, the prefix is put back on the logged message.
func Prefix(prefix string, joinPrefixToMsg bool) WriterAdapterOption {
	return func(a *WriterAdapter) { a.prefix = prefix; a.joinPrefixToMsg = joinPrefixToMsg }
}

// WriterAdapterOption configures a WriterAdapter created by NewWriterAdapter.
type WriterAdapterOption func(*WriterAdapter)

// NewWriterAdapter returns an io.Writer that logs each write through logger.
func NewWriterAdapter(logger log.Logger, options ...WriterAdapterOption) io.Writer {
	adapter := &WriterAdapter{l: logger, msgKey: "msg"}
	for _, option := range options {
		option(adapter)
	}
	return adapter
}
