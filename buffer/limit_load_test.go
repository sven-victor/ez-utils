//go:build !make_test

package buffer

import (
	"bytes"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func init() {
	rand.Seed(time.Now().Unix())
}

func TestLoadLimitWriter(t *testing.T) {
	tests := []struct {
		name        string
		options     []LimitWriterOption
		ignoreError bool
	}{{
		name:        "Test random bytes",
		ignoreError: true,
	}, {
		name:        "Test random bytes with ignore error",
		options:     []LimitWriterOption{LimitWriterIgnoreError},
		ignoreError: false,
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for i := 0; i < 2<<11; i++ {
				writeCount := 0
				w := &bytes.Buffer{}
				lw := LimitWriter(w, int64(i), tt.options...)
				raw := make([]byte, rand.Intn(1024*1024))
				rand.Read(raw)
				for {
					n, err := lw.Write(raw[writeCount:])
					writeCount += n
					if err != nil {
						if !tt.ignoreError {
							require.NoError(t, err)
							break
						}
						break
					}
					if n == 0 {
						break
					}
				}
				var wantWrite string
				gotW := w.String()
				if i > len(raw) {
					wantWrite = string(raw)
				} else {
					wantWrite = string(raw[:i])
				}

				wantWriteLength := len(raw)
				if tt.ignoreError {
					wantWriteLength = len(wantWrite)
				}
				require.Equalf(t, writeCount, wantWriteLength, "Write length does not match expected value.")
				require.Equal(t, gotW, wantWrite)
			}
		})
	}
}
