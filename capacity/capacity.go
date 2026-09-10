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

// Package capacity provides a byte-size type with human-readable parsing and formatting.
package capacity

// Capacities represents a byte capacity in bytes, using binary (1024) units.
type Capacities int64

const (
	// Byte is one byte.
	Byte Capacities = 1
	// Kilobyte is 1024 bytes.
	Kilobyte = Byte << 10
	// Megabyte is 1024 kilobytes.
	Megabyte = Kilobyte << 10
	// Gigabyte is 1024 megabytes.
	Gigabyte = Megabyte << 10
	// Terabyte is 1024 gigabytes.
	Terabyte = Gigabyte << 10
)

// fmtInt formats v into the tail of buf.
// It returns the index where the output begins.
func fmtInt(buf []byte, v uint64) int {
	w := len(buf)
	if v == 0 {
		w--
		buf[w] = '0'
	} else {
		for v > 0 {
			w--
			buf[w] = byte(v%10) + '0'
			v /= 10
		}
	}
	return w
}

var magicUnit = []string{"B", "KB", "MB", "GB", "TB"}

// Set parses s as a capacity and stores the result in c.
// It implements pflag.Value so Capacities can be used as a command-line flag.
func (c *Capacities) Set(s string) error {
	capacities, err := ParseCapacities(s)
	*c = capacities
	return err
}

// Type returns the pflag.Value type name for Capacities ("string").
func (c *Capacities) Type() string {
	return "string"
}

// String formats c as a human-readable binary capacity such as "1GB2MB".
// It implements fmt.Stringer.
func (c *Capacities) String() string {
	var buf [32]byte
	w := len(buf)
	if c == nil {
		return ""
	}
	u := uint64(*c)
	neg := *c < 0
	if neg {
		u = -u
	}

	for _, unit := range magicUnit {
		if u > 0 {
			w -= len(unit)
			for idx, r := range unit {
				buf[w+idx] = byte(r)
			}
			if u%1024 > 0 {
				w = fmtInt(buf[:w], u%1024)
			} else {
				w += len(unit)
			}
			u /= 1024
		} else {
			break
		}
	}

	if neg {
		w--
		buf[w] = '-'
	}
	return string(buf[w:])
}
