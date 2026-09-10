//go:build go1.18

package w

import (
	"encoding/json"
	"fmt"
	"reflect"

	"golang.org/x/exp/slices"
	"gopkg.in/yaml.v3"
)

// Filter returns a new slice containing only the items for which f returns true.
func Filter[T any](old []T, f func(item T) bool) []T {
	newSli := make([]T, 0, len(old))
	for _, item := range old {
		if f(item) {
			newSli = append(newSli, item)
		}
	}
	return newSli
}

// Map applies f to each item in old and returns a new slice of the results.
func Map[S any, T any](old []S, f func(item S) T) []T {
	newSli := make([]T, len(old))
	for idx, item := range old {
		newSli[idx] = f(item)
	}
	return newSli
}

// Has reports whether s contains an element that compare considers equal to t.
func Has[T any](s []T, t T, compare func(a, b T) bool) bool {
	for _, val := range s {
		if ok := compare(val, t); ok {
			return true
		}
	}
	return false
}

// Include reports whether s contains t.
func Include[T comparable](s []T, t T) bool {
	for _, val := range s {
		if val == t {
			return true
		}
	}
	return false
}

// Index returns the index of the first occurrence of t in s, or -1 if t is not present.
func Index[T comparable](s []T, t T) int {
	return slices.Index(s, t)
}

// FindIndex returns the index of the first element that compare considers equal to t, or -1 if none matches.
func FindIndex[T any](s []T, t T, compare func(a, b T) bool) int {
	for idx, val := range s {
		if compare(val, t) {
			return idx
		}
	}
	return -1
}

// Find returns the first element for which f returns true, or the zero value of T if none matches.
func Find[T any](s []T, f func(T) bool) T {
	for _, val := range s {
		if f(val) {
			return val
		}
	}
	return *new(T)
}

// Interfaces converts a slice of T to a slice of interface{}.
func Interfaces[T any](objs []T) []interface{} {
	newObjs := make([]interface{}, len(objs))
	for idx, obj := range objs {
		newObjs[idx] = obj
	}
	return newObjs
}

const (
	// PosRight truncates from the right, keeping the start of the slice.
	PosRight = iota
	// PosLeft truncates from the left, keeping the end of the slice.
	PosLeft
	// PosCenter truncates from the middle, keeping both ends of the slice.
	PosCenter
)

// Limit shortens s to at most limit elements, hiding overflow at hidePosition and appending manySuffix.
func Limit[T any](s []T, limit int, hidePosition int, manySuffix ...T) []T {
	limit = limit - len(manySuffix)
	if len(s) > limit {
		ret := make([]T, 0, limit+len(manySuffix))
		switch hidePosition {
		case PosRight:
			ret = append(ret, s[:limit]...)
			ret = append(ret, manySuffix...)
			return ret
		case PosLeft:
			ret = append(ret, manySuffix...)
			ret = append(ret, s[len(s)-limit:]...)
			return ret
		case PosCenter:
			ret = append(ret, s[:limit/2]...)
			ret = append(ret, manySuffix...)
			ret = append(ret, s[len(s)-(limit-limit/2):]...)
			return ret
		}
	}
	return s
}

// OneOrMore is a slice that marshals as a single value when it has one element, and as an array otherwise.
type OneOrMore[T any] []T

// MarshalYAML implements yaml.Marshaler by encoding a single-element slice as a scalar.
func (s OneOrMore[T]) MarshalYAML() (interface{}, error) {
	if len(s) == 1 {
		return (s)[0], nil
	}
	return []T(s), nil
}

// UnmarshalYAML implements yaml.Unmarshaler, accepting either a YAML scalar or a sequence.
func (s *OneOrMore[T]) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.SequenceNode:
		type plain []T
		return value.Decode((*plain)(s))
	case yaml.ScalarNode:
		var c T
		if err := value.Decode(&c); err != nil {
			return err
		}
		*s = OneOrMore[T]{c}
		return nil
	case yaml.AliasNode:
		return value.Alias.Decode(s)
	}
	return fmt.Errorf("unknown value type: %s", value.Tag)
}

// Equal is implemented by types that can compare themselves to a value of type T.
type Equal[T any] interface {
	// Equal reports whether the receiver is equal to the given value.
	Equal(T) bool
}

// Contains reports whether value is present in the slice.
func (s OneOrMore[T]) Contains(value T) bool {
	for _, item := range s {
		if v, ok := interface{}(value).(Equal[T]); ok {
			return v.Equal(item)
		}
		if reflect.ValueOf(item).Equal(reflect.ValueOf(value)) {
			return true
		}
	}
	return false
}

// UnmarshalJSON implements json.Unmarshaler, accepting either a single JSON value or a JSON array.
func (s *OneOrMore[T]) UnmarshalJSON(data []byte) error {
	var first byte
	if len(data) > 1 {
		first = data[0]
	}

	if first == '[' {
		var parsed []T
		if err := json.Unmarshal(data, &parsed); err != nil {
			return err
		}
		*s = OneOrMore[T](parsed)
		return nil
	}

	var single T
	if err := json.Unmarshal(data, &single); err != nil {
		return err
	}
	switch v := interface{}(single).(type) {
	case T:
		*s = OneOrMore[T]([]T{v})
		return nil
	default:
		return fmt.Errorf("only string or array is allowed, not %T", single)
	}
}

// MarshalJSON implements json.Marshaler, encoding a single-element slice as a JSON value rather than an array.
func (s OneOrMore[T]) MarshalJSON() ([]byte, error) {
	if len(s) == 1 {
		return json.Marshal(s[0])
	}
	return json.Marshal([]T(s))
}

// GroupBy groups items by the key returned from f and collects the corresponding values.
func GroupBy[T any, K comparable, V any](old []T, f func(item T) (groupKey K, value V)) map[K][]V {
	var group = make(map[K][]V)
	for _, item := range old {
		key, value := f(item)
		of := group[key]
		group[key] = append(of, value)
	}
	return group
}
