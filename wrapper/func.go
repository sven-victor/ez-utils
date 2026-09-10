//go:build go1.18

// Package w provides generic helpers for functions, slices, maps, templates, queues, and process groups.
package w

// M unwraps a (value, error) pair and returns the value, or panics if err is non-nil.
func M[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

// E returns the error from a (value, error) pair, discarding the value.
func E[T any](_ T, err error) error {
	return err
}

// P takes the address of o and returns a pointer to a copy of o.
func P[T any](o T) *T {
	return &o
}

// DefaultPointer returns the first non-nil pointer among vals, or nil if all are nil.
func DefaultPointer[T any](vals ...*T) *T {
	for _, val := range vals {
		if val != nil {
			return val
		}
	}
	return nil
}

// DefaultString returns the first non-empty string among vals, or "" if all are empty.
func DefaultString(vals ...string) string {
	for _, val := range vals {
		if len(val) != 0 {
			return val
		}
	}
	return ""
}

// Number is the constraint of integer and floating-point types accepted by DefaultNumber.
type Number interface {
	~int8 | ~int16 | ~int32 | ~int64 |
		~uint8 | ~uint16 | ~uint32 | ~uint64 |
		~float64 | ~float32 | ~int | ~uint
}

// DefaultNumber returns the first non-zero number among vals, or the zero value if all are zero.
func DefaultNumber[T Number](vals ...T) T {
	for _, val := range vals {
		if val != 0 {
			return val
		}
	}
	return 0
}

// Pipe returns a function that applies f to v if e is nil, or returns the zero value of dT and e otherwise.
func Pipe[sT any, dT any](v sT, e error) func(func(sT) (dT, error)) (dT, error) {
	return func(f func(sT) (dT, error)) (dT, error) {
		if e != nil {
			return *new(dT), e
		}
		return f(v)
	}
}

// T is a ternary operator: it returns trueVal if expr is true, and falseVal otherwise.
func T[vT any](expr bool, trueVal, falseVal vT) vT {
	if expr {
		return trueVal
	}
	return falseVal
}
