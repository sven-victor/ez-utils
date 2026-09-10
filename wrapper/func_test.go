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

package w

import (
	"bytes"
	"io"
	"reflect"
	"strconv"
	"testing"
)

func TestPipe(t *testing.T) {
	type testCase[sT any, dT any] struct {
		name     string
		argsFunc func() (sT, error)
		pipeFunc func(sT) (dT, error)
		want     int
		wantErr  bool
	}
	tests := []testCase[[]byte, int]{{
		name: "simple",
		argsFunc: func() ([]byte, error) {
			return io.ReadAll(bytes.NewBufferString("123"))
		},
		pipeFunc: func(i []byte) (int, error) {
			return strconv.Atoi(string(i))
		},
		want: 123,
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Pipe[[]byte, int](tt.argsFunc())(tt.pipeFunc)
			if err != nil && !tt.wantErr {
				t.Errorf("Pipe() = (%v,%v), wantErr %v", got, err, tt.want)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Pipe() = %v, want %v", got, tt.want)
			}
		})
	}
}
