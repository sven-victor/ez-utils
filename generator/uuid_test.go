package g

import (
	"fmt"
	"sort"
	"testing"
	"time"

	"golang.org/x/exp/rand"

	"github.com/stretchr/testify/require"
)

func getSeed(t *testing.T) []string {
	var ret []string
	rand.Seed(uint64(time.Now().UnixMilli()))
	for i := 0; i < rand.Intn(16); i++ {
		maxRead := rand.Intn(1024)
		buf := make([]byte, maxRead)
		_, err := rand.Read(buf)
		require.NoError(t, err)
		ret = append(ret, string(buf))
	}
	return ret
}

func TestNewId(t *testing.T) {
	type args struct {
		seed []string
	}
	tests := []struct {
		name string
		args args
	}{{
		name: "Success",
		args: struct{ seed []string }{seed: getSeed(t)},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var idList []string
			for i := 0; i < 1024; i++ {
				time.Sleep(time.Microsecond)
				idList = append(idList, NewId(tt.args.seed...))
			}
			time.Sleep(time.Second * time.Duration(rand.Intn(3)))
			for _, s := range idList {
				fmt.Println(s)
			}
			require.True(t, sort.StringsAreSorted(idList))
		})
	}
}
