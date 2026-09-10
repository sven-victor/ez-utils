package safe

import (
	"encoding/hex"
	"hash"
)

// Hash is a digest of arbitrary bytes.
type Hash []byte

// HexString returns the hex encoding of h, truncated to n characters when longer.
func (h Hash) HexString(n int) string {
	s := hex.EncodeToString(h)
	if len(s) > n {
		return s[:n]
	}
	return s
}

// NewHash hashes data with hashFunc and returns the digest.
func NewHash(hashFunc func() hash.Hash, data []byte) Hash {
	h := hashFunc()
	h.Write(data)
	return h.Sum(nil)
}
