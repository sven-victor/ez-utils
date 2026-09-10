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

package conv

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMap2Struct(t *testing.T) {
	type testSubStruct struct {
		A string
	}
	type TestStruct struct {
		A int
		B string
		C *testSubStruct
		D testSubStruct
	}
	type args[sT any, dT any] struct {
		src sT
		dst dT
	}
	tests := []struct {
		name    string
		args    args[map[string]interface{}, TestStruct]
		wantErr bool
		want    interface{}
	}{
		{name: "simple", args: args[map[string]interface{}, TestStruct]{src: map[string]interface{}{"A": 1, "B": "AA"}, dst: TestStruct{}}, want: TestStruct{A: 1, B: "AA"}},
		{
			name: "sub struct",
			args: args[map[string]interface{}, TestStruct]{
				src: map[string]interface{}{
					"A": 1,
					"C": map[string]string{"A": "xx"},
					"D": map[string]string{"A": "cc"},
				},
				dst: TestStruct{},
			},
			want: TestStruct{A: 1, C: &testSubStruct{A: "xx"}, D: testSubStruct{A: "cc"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := JSON(tt.args.src, &tt.args.dst); (err != nil) != tt.wantErr {
				t.Errorf("JSON() error = %v, wantErr %v", err, tt.wantErr)
			} else if !tt.wantErr {
				require.Equal(t, tt.want, tt.args.dst)
			}
		})
	}
}

func TestStruct2Struct(t *testing.T) {
	type testSubStructA struct {
		A string
	}
	type TestStructA struct {
		A int
		B string
		C *testSubStructA
		D testSubStructA
		E string
	}
	type testSubStructB struct {
		A string
		F string
	}
	type TestStructB struct {
		A int
		B string
		C testSubStructB
		D *testSubStructB
		F string
	}
	type args[sT any, dT any] struct {
		src sT
		dst dT
	}
	tests := []struct {
		name    string
		args    args[TestStructA, TestStructB]
		wantErr bool
		want    TestStructB
	}{
		{name: "simple", args: args[TestStructA, TestStructB]{src: TestStructA{A: 1, B: "AA"}, dst: TestStructB{}}, want: TestStructB{A: 1, B: "AA", D: &testSubStructB{}}},
		{
			name: "sub struct",
			args: args[TestStructA, TestStructB]{
				src: TestStructA{
					A: 1,
					C: &testSubStructA{A: "xx"},
					D: testSubStructA{A: "cc"},
					E: "AAA",
				},
				dst: TestStructB{F: "ABC", D: &testSubStructB{F: "XYZ"}},
			},
			want: TestStructB{
				A: 1,
				C: testSubStructB{A: "xx"},
				D: &testSubStructB{A: "cc", F: "XYZ"},
				F: "ABC",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := JSON(tt.args.src, &tt.args.dst); (err != nil) != tt.wantErr {
				t.Errorf("JSON() error = %v, wantErr %v", err, tt.wantErr)
			} else if !tt.wantErr {
				require.Equal(t, tt.want, tt.args.dst)
			}
		})
	}
}

func TestStruct2StructError(t *testing.T) {
	type testSubStructA struct {
		A string
	}
	type TestStructA struct {
		A int
		B string
		C *testSubStructA
		D testSubStructA
		E string
		G string
	}
	type testSubStructB struct {
		A string
		F string
	}
	type TestStructB struct {
		A int
		B string
		C testSubStructB
		D *testSubStructB
		F string
		G int
	}
	type args[sT any, dT any] struct {
		src sT
		dst dT
	}
	tests := []struct {
		name    string
		args    args[TestStructA, TestStructB]
		wantErr bool
		want    TestStructB
	}{
		{name: "simple", args: args[TestStructA, TestStructB]{src: TestStructA{A: 1, B: "AA"}, dst: TestStructB{}}, want: TestStructB{A: 1, B: "AA", D: &testSubStructB{}}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := JSON(tt.args.src, &tt.args.dst); (err != nil) != tt.wantErr {
				t.Errorf("JSON() error = %v, wantErr %v", err, tt.wantErr)
			} else if !tt.wantErr {
				require.Equal(t, tt.want, tt.args.dst)
			}
		})
	}
}
