// Package conv provides helpers for converting values between common encodings
// such as JSON, time formats, and URL query strings.
package conv

import (
	"encoding/json"
)

// JSON marshals src to JSON and unmarshals the result into dst.
// dst must be a non-nil pointer.
func JSON[sT any, dT any](src sT, dst *dT) error {
	bytes, err := json.Marshal(src)
	if err != nil {
		return err
	}
	return json.Unmarshal(bytes, dst)
}
