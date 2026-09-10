package w

import (
	"fmt"
	"testing"
	"time"

	kitlog "github.com/go-kit/log"
	"github.com/stretchr/testify/require"

	"github.com/sven-victor/ez-utils/signals"
)

func TestGroup_Run(t *testing.T) {
	ExitFunc = func(code int) {
		require.Equal(t, code, 0)
	}
	signals.SetupSignalHandler(kitlog.NewNopLogger())
	var g Group
	start := time.Now()
	timer1 := time.NewTimer(time.Second * 1)
	g.Add(func() error {
		<-timer1.C
		return fmt.Errorf("auto stop 1")
	}, func(err error) {
		timer1.Reset(0)
		require.ErrorContains(t, err, "auto stop 1")
	})
	timer2 := time.NewTimer(time.Second * 3)
	g.Add(func() error {
		<-timer2.C
		return fmt.Errorf("auto stop 2")
	}, func(err error) {
		require.ErrorContains(t, err, "auto stop 1")
		timer2.Reset(0)
		time.Sleep(time.Second)
	})
	require.ErrorContains(t, g.Run(), "auto stop 1")
	end := time.Now()
	dur := end.Sub(start)
	if dur < time.Second*2 || dur > time.Second*21/10 {
		t.Error("Abnormal duration")
	}
}
