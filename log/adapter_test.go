package log

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAdapterMessageKey(t *testing.T) {
	var w bytes.Buffer
	testLogger := New(WithWriter(&w))
	{
		_, err := NewWriterAdapter(testLogger, MessageKey("qwerty")).Write([]byte("zxcv"))
		require.NoError(t, err)
		ok := strings.HasSuffix(w.String(), "qwerty=zxcv\n")
		require.Truef(t, ok, "The log output is inconsistent with the expected: log=%s", w.String())
	}
}

func TestAdapterPrefix(t *testing.T) {
	type args struct {
		prefix          string
		joinPrefixToMsg bool
		msg             string
	}
	tests := []struct {
		name                string
		args                args
		wantOutputLogSuffix string
	}{
		{name: "Test don't join prefix to msg and msg not container prefix", args: args{joinPrefixToMsg: false, prefix: "qwerty:", msg: "abc"}, wantOutputLogSuffix: " msg=abc\n"},
		{name: "Test join prefix to msg and msg not container prefix", args: args{joinPrefixToMsg: true, prefix: "qwerty:", msg: "abc"}, wantOutputLogSuffix: " msg=qwerty:abc\n"},
		{name: "Test don't join prefix to msg and msg container prefix", args: args{joinPrefixToMsg: false, prefix: "qwerty:", msg: "qwerty:abc"}, wantOutputLogSuffix: " msg=abc\n"},
		{name: "Test join prefix to msg and msg container prefix", args: args{joinPrefixToMsg: true, prefix: "qwerty:", msg: "qwerty:abc"}, wantOutputLogSuffix: " msg=qwerty:abc\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var w bytes.Buffer
			testLogger := New(WithWriter(&w))
			_, err := NewWriterAdapter(testLogger, Prefix(tt.args.prefix, tt.args.joinPrefixToMsg)).Write([]byte(tt.args.msg))
			require.NoError(t, err)
			ok := strings.HasSuffix(w.String(), tt.wantOutputLogSuffix)
			require.Truef(t, ok, "The log output is inconsistent with the expected: log=%s,wantSuffix=%s", w.String(), tt.wantOutputLogSuffix)
		})
	}
}
