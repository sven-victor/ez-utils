//go:build !make_test

package buffer

import (
	"bytes"
	"io"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func init() {
	rand.Seed(time.Now().Unix())
}

func TestLoadNewPreReader(t *testing.T) {
	t.Run("Test random data", func(t *testing.T) {
		for i := 0; i < 2<<10; i++ {
			raw := make([]byte, rand.Intn(1024*1024))
			rand.Read(raw)
			r := io.NopCloser(bytes.NewBuffer(raw))
			pr, err := NewPreReader(r, i)
			require.NoError(t, err)
			var wantPreBuf []byte
			if len(raw) < i {
				wantPreBuf = raw
			} else {
				wantPreBuf = raw[:i]
			}
			require.Equal(t, pr.String(), string(wantPreBuf))
		}
	})
}
