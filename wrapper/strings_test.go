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
	"reflect"
	"testing"
)

func TestJSONStringer(t *testing.T) {
	type args struct {
		v any
	}
	tests := []struct {
		name string
		args args
		want string
	}{{
		name: "test int",
		args: args{v: 1},
		want: "1",
	}, {
		name: "test float",
		args: args{v: 1.99},
		want: "1.99",
	}, {
		name: "test string",
		args: args{v: "hello"},
		want: `"hello"`,
	}, {
		name: "test struct",
		args: args{v: struct {
			FieldA string  `json:"field_a"`
			FieldB int     `json:"field-b"`
			FieldC float64 `json:"fieldC"`
			FieldD []byte
			fieldE string
		}{FieldA: "hello", FieldB: 123, FieldC: -22.3, FieldD: []byte("world"), fieldE: "bbb"}},
		want: `{"field_a":"hello","field-b":123,"fieldC":-22.3,"FieldD":"d29ybGQ="}`,
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := JSONStringer(tt.args.v).String(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("JSONStringer() = %v, want %v", got, tt.want)
			}
		})
		t.Run(tt.name+":stringer", func(t *testing.T) {
			if got := NewStringer(JSONStringer(tt.args.v).String).String(); !StringEqual(got, tt.want) {
				t.Errorf("NewStringer() = %v, want %v", got, tt.want)
			}
		})
	}
}
