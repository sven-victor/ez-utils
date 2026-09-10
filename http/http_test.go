package http

import (
	"net/http"
	"testing"

	"github.com/sven-victor/ez-utils/sets"
	w "github.com/sven-victor/ez-utils/wrapper"
)

func TestJoinPath(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{{
		name: "",
		args: []string{"/", "aa", "bb", "cc"},
		want: "/aa/bb/cc",
	}, {
		name: "",
		args: []string{"aa", "bb", "cc"},
		want: "/aa/bb/cc",
	}, {
		name: "",
		args: []string{"/aa/", "/bb/"},
		want: "/aa/bb/",
	}, {
		name: "",
		args: []string{"/aa", "/bb/", "cc"},
		want: "/aa/bb/cc",
	}, {
		name: "",
		args: []string{"/aa", "../", "../cc", "dd/ee"},
		want: "/aa/../../cc/dd/ee",
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := JoinPath(tt.args...); got != tt.want {
				t.Errorf("JoinPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetRemoteAddr(t *testing.T) {
	type args struct {
		r       *http.Request
		trustIP sets.IPNets
	}
	tests := []struct {
		name string
		args args
		want string
	}{{
		name: "simple",
		args: args{
			r: &http.Request{Header: map[string][]string{"X-Forwarded-For": {"192.168.1.1,123.222.123.3,192.168.1.1,10.0.0.1,1.1.1.1"}}},
		},
		want: "1.1.1.1",
	}, {
		name: "trustCidr",
		args: args{
			r:       &http.Request{Header: map[string][]string{"X-Forwarded-For": {"192.168.1.1,123.222.123.3,192.168.1.1,10.0.0.1,1.1.1.1"}}},
			trustIP: []sets.IPNet{w.M(sets.ParseIPNet("1.1.1.1"))},
		},
		want: "123.222.123.3",
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetRemoteAddr(tt.args.r, tt.args.trustIP); got != tt.want {
				t.Errorf("GetRemoteAddr() = %v, want %v", got, tt.want)
			}
		})
	}
}
