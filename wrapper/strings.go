package w

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// Stringer wraps a function as an fmt.Stringer.
type Stringer struct {
	stringer func() string
}

// String implements fmt.Stringer by calling the wrapped function.
func (s Stringer) String() string {
	return s.stringer()
}

// NewStringer returns an fmt.Stringer that calls stringer to produce its string value.
func NewStringer(stringer func() string) fmt.Stringer {
	return &Stringer{stringer}
}

// StringEqual reports whether a and b are equal.
func StringEqual(a, b string) bool {
	return a == b
}

type jsonStringer struct {
	v interface{}
}

func (j jsonStringer) String() string {
	buf := bytes.Buffer{}
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(j.v); err != nil {
		return ""
	}
	return strings.TrimSpace(buf.String())
}

// JSONStringer returns an fmt.Stringer that encodes v as JSON.
func JSONStringer(v any) fmt.Stringer {
	return &jsonStringer{v: v}
}
