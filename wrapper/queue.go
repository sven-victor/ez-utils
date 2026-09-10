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
	"container/ring"
	"errors"
	"sync"
)

var (
	// QueueFull is returned by RingQueue.Put when the queue has no remaining capacity.
	QueueFull = errors.New("queue is full") //nolint:revive
	// QueueNull is returned by RingQueue.Get when the queue is empty.
	QueueNull = errors.New("queue is null") //nolint:revive
)

// RingQueue is a concurrent, fixed-capacity ring buffer.
type RingQueue[T any] struct {
	r               *ring.Ring
	start, end, max int
	mux             sync.Mutex
}

// NewRingQueue returns a RingQueue that can hold at most n elements.
func NewRingQueue[T any](n int) *RingQueue[T] {
	return &RingQueue[T]{
		r:     ring.New(n + 1),
		start: 0, end: 0, max: n + 1,
	}
}

// Do calls f for each element currently in the queue, from front to back.
func (r *RingQueue[T]) Do(f func(a T)) {
	r.mux.Lock()
	defer r.mux.Unlock()
	start := r.r.Move(r.start)
	end := r.r.Move(r.end)
	for ; start != end; start = start.Next() {
		f(start.Value.(T))
	}
}

// List returns a snapshot of the elements currently in the queue, from front to back.
func (r *RingQueue[T]) List() []T {
	r.mux.Lock()
	defer r.mux.Unlock()
	var items []T
	start := r.r.Move(r.start)
	end := r.r.Move(r.end)
	for ; start != end; start = start.Next() {
		items = append(items, start.Value.(T))
	}
	return items
}

// Remove deletes every element for which f returns true.
func (r *RingQueue[T]) Remove(f func(a T) bool) {
	r.mux.Lock()
	defer r.mux.Unlock()

	start := r.r.Move(r.start)
	end := r.r.Move(r.end)
	isPrev := false
	for ; start != end; start = start.Next() {
		if start == r.r {
			isPrev = true
		}
		if f(start.Value.(T)) {
			end.Link(ring.New(1))
			if r.start > r.end && !isPrev {
				r.start++
				if r.start >= r.max {
					r.start -= r.max
				}
			} else {
				r.end--
				if r.end < 0 {
					r.end += r.max
				}
			}
			prev := start.Prev()
			prev.Link(start.Next())
			start = prev
		}
	}
}

// Put appends val to the queue. It returns QueueFull if the queue has no remaining capacity.
func (r *RingQueue[T]) Put(val T) error {
	r.mux.Lock()
	defer r.mux.Unlock()
	if r.start == r.end+1 {
		return QueueFull
	} else if r.end+1 == r.max && r.start == 0 {
		return QueueFull
	}
	r.r.Move(r.end).Value = val
	r.end++
	if r.end == r.max {
		r.end = 0
	}
	return nil
}

// Get removes and returns the front element. It returns QueueNull if the queue is empty.
func (r *RingQueue[T]) Get() (T, error) {
	if r.start == r.end {
		return *new(T), QueueNull
	}
	r.mux.Lock()
	defer r.mux.Unlock()
	val := r.r.Move(r.start).Value.(T)
	r.r.Move(r.start).Value = nil
	r.start++
	if r.start == r.max {
		r.start = 0
	}
	return val, nil
}

// Len reports the number of elements currently in the queue.
func (r *RingQueue[T]) Len() int {
	if r.start < r.end {
		return r.end - r.start
	} else if r.start == r.end {
		return 0
	} else {
		return r.end + r.max - r.start
	}
}
