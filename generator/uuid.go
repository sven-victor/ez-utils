// Package g generates time-prefixed UUIDs and trace-style identifiers.
package g

import (
	"encoding/binary"
	"time"

	uuid "github.com/satori/go.uuid"
	"go.opentelemetry.io/otel/trace"
)

const midUint64 = uint64(1) << 63

// NewId returns a 32-character hex identifier derived from NewUUID.
func NewId(seed ...string) string {
	return trace.TraceID(NewUUID(seed...)).String()
}

// NewUUID returns a UUID v4 whose first eight bytes are a microsecond timestamp
// optionally mixed with a hash of seed, so IDs sort roughly by creation time.
func NewUUID(seed ...string) uuid.UUID {
	ts := uint64(time.Now().UnixMicro())
	if ts < midUint64 {
		var hash uint64
		for _, s := range seed {
			seedBytes := []byte(s)
			for i := 0; i < len(seedBytes); i += 8 {
				if len(seedBytes[i:]) <= 8 {
					tmp := make([]byte, 8)
					copy(tmp, seedBytes[i:])
					hash += binary.BigEndian.Uint64(tmp)
					break
				}
				hash += binary.BigEndian.Uint64(seedBytes[i : i+8])
			}
		}
		if hash > midUint64 {
			hash = hash - midUint64
		}
		ts += hash
	}

	id := uuid.Must(uuid.NewV4())
	binary.BigEndian.PutUint64(id[:8], ts)
	return id
}
