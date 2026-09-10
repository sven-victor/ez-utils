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

package sets

import (
	"fmt"
	"reflect"
	"sync/atomic"
)

// ArraySet is a fixed-capacity collection of unique values stored in insertion order.
type ArraySet[T any] struct {
	arr          []T
	cur          int64
	size         int64
	fullCallback func(arr []T, length int64) error
}

// NewArraySet returns an ArraySet with the given capacity, then appends val.
func NewArraySet[T any](size int64, val ...T) (*ArraySet[T], error) {
	s := ArraySet[T]{
		arr:  make([]T, size),
		size: size,
		cur:  -1,
	}
	return &s, s.Append(val...)
}

// SetFullCallback registers a function invoked when Append would exceed the set capacity.
func (s *ArraySet[T]) SetFullCallback(call func([]T, int64) error) {
	s.fullCallback = call
}

// CallFullCallback runs the registered full callback, if any.
func (s *ArraySet[T]) CallFullCallback() error {
	if s.fullCallback != nil {
		return s.fullCallback(s.arr, s.cur+1)
	}
	return nil
}

// Reset removes all elements while keeping the original capacity.
func (s *ArraySet[T]) Reset() {
	s.arr = make([]T, s.size)
	s.cur = -1
}

// ErrorSetFull is returned by Append when the set is at capacity and no full callback is set.
var ErrorSetFull = fmt.Errorf("set Is Full")

// Append adds unique values to the set. Duplicate values are ignored.
func (s *ArraySet[T]) Append(values ...T) error {
loop:
	for _, value := range values {
		for _, val := range s.arr {
			if compare(val, value) {
				continue loop
			}
		}
		idx := atomic.AddInt64(&s.cur, 1)
		if idx >= s.size {
			atomic.AddInt64(&s.cur, -1)
			if s.fullCallback != nil {
				if err := s.fullCallback(s.arr, s.cur+1); err != nil {
					return err
				} else if idx = atomic.AddInt64(&s.cur, 1); idx >= s.size {
					atomic.AddInt64(&s.cur, -1)
					return nil
				}
			} else {
				return ErrorSetFull
			}
		}
		s.arr[idx] = value
	}
	return nil
}

// Index returns the index of value, or -1 if it is not present.
func (s *ArraySet[T]) Index(value T) int {
	for idx, val := range s.arr {
		if compare(val, value) {
			return idx
		}
	}
	return -1
}

// Include reports whether value is already in the set.
func (s *ArraySet[T]) Include(value T) bool {
	for _, val := range s.arr {
		if compare(val, value) {
			return true
		}
	}
	return false
}

// Size returns the maximum number of elements the set can hold.
func (s *ArraySet[T]) Size() int64 {
	return s.size
}

// Length returns the current number of elements.
func (s *ArraySet[T]) Length() int64 {
	return s.cur + 1
}

// List returns the stored elements in insertion order.
func (s *ArraySet[T]) List() []T {
	return s.arr[:s.cur+1]
}

func compare(a interface{}, b interface{}) bool {
	switch a.(type) {
	case int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64, string, bool:
		switch b.(type) {
		case int, int8, int16, int32, int64,
			uint, uint8, uint16, uint32, uint64,
			float32, float64, string, bool:
			return a == b
		}
	}
	if reflect.TypeOf(a).Kind() == reflect.TypeOf(b).Kind() {
		return reflect.DeepEqual(a, b)
	}
	return false
}
