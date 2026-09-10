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
