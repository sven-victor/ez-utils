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

// Package crypto provides AES cipher helpers for CBC encryption and decryption.
package crypto

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
)

// AesKey is the set of AES key lengths: 16 bytes (AES-128), 24 bytes (AES-192),
// or 32 bytes (AES-256).
type AesKey interface {
	[16]byte | [24]byte | [32]byte
}

// AESCipher wraps an AES block cipher and exposes CBC encrypt and decrypt helpers.
type AESCipher struct {
	err   error
	block cipher.Block
	key   []byte
}

// NewAESCipher constructs an AESCipher from key. Key length must be 16, 24, or 32 bytes.
// Construction errors are returned from later CBCEncrypt/CBCDecrypt calls rather than from NewAESCipher.
func NewAESCipher(key []byte) *AESCipher {
	c := AESCipher{key: key}
	c.block, c.err = aes.NewCipher(key)
	return &c
}

// CBCEncrypt encrypts origData with AES-CBC and PKCS#7 padding. The IV is the
// first blockSize bytes of the key.
func (c AESCipher) CBCEncrypt(origData []byte) ([]byte, error) {
	if c.err != nil {
		return nil, c.err
	}
	blockSize := c.block.BlockSize()
	origData = pkcs7Padding(origData, blockSize)
	blockMode := cipher.NewCBCEncrypter(c.block, c.key[:blockSize])
	cryted := make([]byte, len(origData))
	blockMode.CryptBlocks(cryted, origData)
	return cryted, nil
}

// CBCDecrypt decrypts AES-CBC ciphertext produced by CBCEncrypt and removes PKCS#7 padding.
func (c AESCipher) CBCDecrypt(crytedByte []byte) ([]byte, error) {
	if c.err != nil {
		return nil, c.err
	}
	blockSize := c.block.BlockSize()
	blockMode := cipher.NewCBCDecrypter(c.block, c.key[:blockSize])
	orig := make([]byte, len(crytedByte))
	blockMode.CryptBlocks(orig, crytedByte)
	orig = pkcs7UnPadding(orig)
	return orig, nil
}

func pkcs7Padding(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padtext...)
}

func pkcs7UnPadding(origData []byte) []byte {
	length := len(origData)
	unpadding := int(origData[length-1])
	return origData[:(length - unpadding)]
}
