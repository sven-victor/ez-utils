package log

import (
	"bytes"
	"strings"
	"testing"

	"github.com/go-kit/log/level"
	"github.com/stretchr/testify/require"
)

func TestWithPrint(t *testing.T) {
	var w bytes.Buffer
	testLogger := New(WithWriter(&w))
	{
		withPrintLogger := WithPrint("Hello1")(testLogger)
		withPrintLogger = WithPrint("Hello2")(withPrintLogger)
		if err := level.Error(withPrintLogger).Log("hello", "world"); err != nil {
			t.Fatal(err)
		}
		ok := strings.HasSuffix(w.String(), "hello=world\nHello1\nHello2\n")
		require.Truef(t, ok, "The log output is inconsistent with the expected: log=%s", w.String())
	}
}
