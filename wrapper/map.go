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

import "sort"

// Comparable is the constraint of ordered types that can be used as SortedMap keys.
type Comparable interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 |
		~float32 | ~float64 | ~string
}

// SortedMapItem is a key-value pair stored in a SortedMap.
type SortedMapItem[K Comparable, V any] struct {
	// Key is the map key used for sorting.
	Key K
	// Value is the value associated with Key.
	Value V
}

// SortedMap is a slice of key-value pairs sorted by key.
type SortedMap[K Comparable, V any] []SortedMapItem[K, V]

// Len reports the number of items in the sorted map.
func (s SortedMap[K, V]) Len() int { return len(s) }

// Less reports whether the item at i should sort before the item at j.
func (s SortedMap[K, V]) Less(i, j int) bool { return s[i].Key < s[j].Key }

// Swap exchanges the items at i and j.
func (s SortedMap[K, V]) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

// NewSortedMap builds a SortedMap from one or more maps and sorts it by key.
func NewSortedMap[K Comparable, V any](kvs ...map[K]V) SortedMap[K, V] {
	m := SortedMap[K, V]{}
	for _, kvMap := range kvs {
		for k, v := range kvMap {
			m = append(m, SortedMapItem[K, V]{Key: k, Value: v})
		}
	}
	sort.Sort(m)
	return m
}

// Merge combines maps into a single map. Later maps overwrite earlier keys.
func Merge[M ~map[K]V, K comparable, V any](maps ...M) M {
	fullCap := 0
	for _, m := range maps {
		fullCap += len(m)
	}

	merged := make(M, fullCap)
	for _, m := range maps {
		for key, val := range m {
			merged[key] = val
		}
	}

	return merged
}
