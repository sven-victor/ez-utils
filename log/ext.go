package log

import (
	"fmt"
	"io"

	"github.com/go-kit/log"
)

type withPrint struct{}

// WithPrint returns an Option that writes raw to the logger's writer after each log record.
func WithPrint(raw interface{}) Option {
	return func(l log.Logger) log.Logger {
		return log.With(l, withPrint{}, raw)
	}
}

type extLogger struct {
	logger log.Logger
	w      io.Writer
}

func writeln(w io.Writer, v []byte) {
	if len(v) > 0 {
		_, _ = w.Write(v)
		if v[len(v)-1] != '\n' {
			_, _ = w.Write([]byte{'\n'})
		}
	}
}

func (l extLogger) Log(keyvals ...interface{}) error {
	if len(keyvals)%2 != 0 {
		keyvals = append(keyvals, log.ErrMissingValue)
	}
	var values []interface{}
	callerIndex := -1
	for i := 0; i < len(keyvals); {
		key := keyvals[i]
		val := keyvals[i+1]
		switch key.(type) {
		case callerName:
			if callerIndex < 0 {
				callerIndex = i
			} else {
				keyvals[callerIndex+1] = val
				keyvals = append(keyvals[:i], keyvals[i+2:]...)
				continue
			}
		case withPrint:
			values = append(values, val)
			keyvals = append(keyvals[:i], keyvals[i+2:]...)
			continue
		}
		i += 2
	}
	defer func() {
		for _, value := range values {
			switch v := value.(type) {
			case string:
				writeln(l.w, []byte(v))
			case fmt.Stringer:
				writeln(l.w, []byte(v.String()))
			default:
				if value != nil {
					writeln(l.w, []byte(fmt.Sprintf("%#v", value)))
				}
			}
		}
	}()

	return l.logger.Log(keyvals...)
}
