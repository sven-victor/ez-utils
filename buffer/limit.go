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

// Package buffer provides I/O wrappers that limit how much data is written
// and a reader that prefetches a prefix of a stream for later inspection.
package buffer

import (
	"io"
)

// LimitWriterOption configures a LimitedWriter returned by LimitWriter.
type LimitWriterOption func(*LimitedWriter)

// LimitWriter returns a Writer that writes to w
// but stops with EOF after n bytes.
// The underlying implementation is a *LimitedWriter.
func LimitWriter(w io.Writer, n int64, options ...LimitWriterOption) io.Writer {
	writer := &LimitedWriter{W: w, N: n, MaxN: n}
	for _, option := range options {
		option(writer)
	}
	return writer
}

// LimitWriterIgnoreError is a LimitWriterOption that makes Write ignore errors
// and always report that the full input was written.
func LimitWriterIgnoreError(lw *LimitedWriter) {
	lw.ignoreError = true
}

// LimitedWriter writes to W but limits the amount of
// data written to just N bytes. Each call to Write
// updates N to reflect the new amount remaining.
// Write returns EOF when N <= 0.
type LimitedWriter struct {
	// W is the underlying writer.
	W io.Writer
	// N is the number of bytes remaining that may be written.
	N int64
	// MaxN is the original write limit, in bytes.
	MaxN        int64
	ignoreError bool
}

// Write implements io.Writer. It writes at most l.N remaining bytes to l.W.
// When no capacity remains, it returns io.EOF unless LimitWriterIgnoreError was
// applied, in which case Write reports success and the original buffer length.
func (l *LimitedWriter) Write(p []byte) (n int, err error) {
	c := len(p)
	if l.N <= 0 {
		err = io.EOF
	} else {
		if int64(len(p)) > l.N {
			p = p[0:l.N]
		}
		n, err = l.W.Write(p)
		l.N -= int64(n)
	}
	if l.ignoreError {
		return c, nil
	}
	return
}
