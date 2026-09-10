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

// Package storage defines a backend-agnostic object storage interface and a
// registry for constructing storage clients from configuration.
package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	stdtime "time"

	"github.com/spf13/cast"
)

// Storage is an object store that also implements fs.ReadDirFS and json.Marshaler.
// Register implementations with RegisterStorage and construct them with NewClient.
type Storage interface {
	fs.ReadDirFS
	json.Marshaler
	Type() string
	Name() string
	ListObject(ctx context.Context, objectPrefix string, recursion bool, callback func(key Object)) error
	HeadObject(ctx context.Context, objectPath string) (obj *Object, err error)
	GetObject(ctx context.Context, objectPath string) (*ObjectReader, error)
	PutObject(ctx context.Context, objectPath string, obj io.Reader, headers http.Header, metadata map[string]string) error
}

// Object is a stored object or directory entry. Object implements fs.DirEntry.
type Object struct {
	Key          string
	LastModified stdtime.Time
	Size         int64
	ETag         string
	StorageClass string
	Mode         fs.FileMode
	Headers      http.Header
	Metadata     map[string]string
}

// Name implements fs.DirEntry.
func (o Object) Name() string {
	return o.Key
}

// IsDir implements fs.DirEntry.
func (o Object) IsDir() bool {
	return o.Mode.IsDir()
}

// Type implements fs.DirEntry.
func (o Object) Type() fs.FileMode {
	return o.Mode
}

// Info implements fs.DirEntry.
func (o Object) Info() (fs.FileInfo, error) {
	return &FileInfo{
		name:    o.Key,
		size:    o.Size,
		mode:    o.Mode,
		modTime: o.LastModified,
	}, nil
}

// ObjectReader is an Object with an open body. ObjectReader implements fs.File.
type ObjectReader struct {
	io.ReadCloser
	Object
}

// FileInfo implements fs.FileInfo for a stored object.
type FileInfo struct {
	name    string
	size    int64
	mode    fs.FileMode
	modTime stdtime.Time
}

// Name implements fs.FileInfo.
func (f FileInfo) Name() string {
	return f.name
}

// Size implements fs.FileInfo.
func (f FileInfo) Size() int64 {
	return f.size
}

// Mode implements fs.FileInfo.
func (f FileInfo) Mode() fs.FileMode {
	return f.mode
}

// ModTime implements fs.FileInfo.
func (f FileInfo) ModTime() stdtime.Time {
	return f.modTime
}

// IsDir implements fs.FileInfo.
func (f FileInfo) IsDir() bool {
	return f.mode.IsDir()
}

// Sys implements fs.FileInfo.
func (f FileInfo) Sys() any {
	return nil
}

// Stat implements fs.File.
func (o ObjectReader) Stat() (fs.FileInfo, error) {
	return o.Info()
}

// RawPermissionsToMode parses a Unix-style permission string such as
// "-rw-r--r--" or "rwxr-xr-x" into an fs.FileMode. Invalid input yields 0o777.
func RawPermissionsToMode(raw string) fs.FileMode {
	const str = "dalTLDpSugct?"
	if len(raw) == 9 {
		raw = "-" + raw
	}
	if len(raw) != 10 {
		return 0o777
	}
	permissions := []rune(raw)
	var mode fs.FileMode
	if raw[0] != '-' {
		for i, c := range str {
			if permissions[0] == c {
				mode |= 1 << uint(32-1-i)
			}
		}
	}
	rwx := []rune("rwxrwxrwx")
	for i, ch := range permissions[1:] {
		if rwx[i] == ch {
			mode |= 1 << uint(8-i)
		}
	}

	return mode
}

// ConfigProvider reads string and integer settings used to construct a Storage.
type ConfigProvider interface {
	GetString(name string) string
	GetInt(name string) int
}

type mapConfigProvider map[string]interface{}

func (v mapConfigProvider) GetString(key string) string {
	return cast.ToString(v.Get(key))
}

func (v mapConfigProvider) Get(key string) interface{} {
	val, _ := v[key]
	return val
}

func (v mapConfigProvider) GetInt(key string) int {
	return cast.ToInt(v.Get(key))
}

// NewMapConfigProvider wraps m as a ConfigProvider. Keys are looked up with
// GetString and GetInt using spf13/cast conversions.
func NewMapConfigProvider(m map[string]interface{}) ConfigProvider {
	return mapConfigProvider(m)
}

var registeredStorage = make(map[string]func(ctx context.Context, v ConfigProvider) (Storage, error))

// RegisterStorage registers newFunc under storageType for NewClient.
// It panics if storageType is already registered.
func RegisterStorage(storageType string, newFunc func(ctx context.Context, v ConfigProvider) (Storage, error)) {
	if _, ok := registeredStorage[storageType]; ok {
		panic("Duplicate registration of storage types")
	}
	registeredStorage[storageType] = newFunc
}

// NewClient constructs a Storage from v. The "type" key selects a backend
// registered with RegisterStorage.
func NewClient(ctx context.Context, v ConfigProvider) (Storage, error) {
	storageType := v.GetString("type")
	if storageType == "" {
		return nil, fmt.Errorf("storage type is empty")
	}
	if newFunc, ok := registeredStorage[storageType]; ok {
		return newFunc(ctx, v)
	}
	return nil, fmt.Errorf("unknown backend type %s", storageType)
}
