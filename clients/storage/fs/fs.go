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

// Package fs implements storage.Storage over local files, zip archives, and
// in-memory filesystems.
package fs

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/aws/smithy-go/time"
	"github.com/go-kit/log/level"
	"github.com/spf13/afero"

	"github.com/sven-victor/ez-utils/clients/storage"
	"github.com/sven-victor/ez-utils/log"
)

// Options configures a filesystem storage client.
type Options struct {
	// Base is the root directory or zip file path, depending on Type.
	Base string `json:"base,omitempty" yaml:"base,omitempty"`
	// Type is the backend kind: local, file, zip, in-memory, or inmemory.
	Type string `json:"type,omitempty" yaml:"type,omitempty"`
}

// Client implements storage.Storage and fs.FS using an io/fs filesystem.
type Client struct {
	fs     fs.FS
	fsType string
	o      Options
}

// ReadDir implements fs.ReadDirFS and storage.Storage. file:// and local://
// prefixes are stripped from name.
func (c Client) ReadDir(name string) ([]fs.DirEntry, error) {
	if strings.HasPrefix(name, "file://") {
		name = strings.TrimPrefix(name, "file://")
	} else if strings.HasPrefix(name, "local://") {
		name = strings.TrimPrefix(name, "local://")
	}
	return fs.ReadDir(c.fs, name)
}

// Open implements fs.FS and storage.Storage by opening the named object.
func (c Client) Open(name string) (fs.File, error) {
	return c.GetObject(context.Background(), name)
}

// MarshalJSON implements json.Marshaler and storage.Storage.
func (c Client) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.o)
}

// Name implements storage.Storage. It returns type://base.
func (c Client) Name() string {
	return fmt.Sprintf("%s://%s", c.Type(), c.o.Base)
}

// GetObject implements storage.Storage. file:// and local:// prefixes are
// stripped from objectPath.
func (c Client) GetObject(_ context.Context, objectPath string) (*storage.ObjectReader, error) {
	if strings.HasPrefix(objectPath, "file://") {
		objectPath = strings.TrimPrefix(objectPath, "file://")
	} else if strings.HasPrefix(objectPath, "local://") {
		objectPath = strings.TrimPrefix(objectPath, "local://")
	}
	f, err := c.fs.Open(objectPath)
	if err != nil {
		return nil, err
	}
	stat, err := f.Stat()
	if err != nil {
		return nil, err
	}
	return &storage.ObjectReader{
		ReadCloser: f,
		Object: storage.Object{
			LastModified: stat.ModTime(),
			Size:         stat.Size(),
			StorageClass: c.Type(),
			Metadata:     map[string]string{},
			Mode:         stat.Mode(),
			Headers: map[string][]string{
				"Content-Length": {strconv.FormatInt(stat.Size(), 10)},
				"Last-Modified":  {time.FormatDateTime(stat.ModTime())},
			},
		},
	}, nil
}

// OpenFileFS is an optional filesystem that supports opening files for writing.
type OpenFileFS interface {
	OpenFile(name string, flag int, perm os.FileMode) (*os.File, error)
}

// PutObject implements storage.Storage. The underlying FS must implement
// OpenFileFS. file:// and local:// prefixes are stripped from objectPath.
func (c Client) PutObject(ctx context.Context, objectPath string, obj io.Reader, headers http.Header, metadata map[string]string) error {
	if strings.HasPrefix(objectPath, "file://") {
		objectPath = strings.TrimPrefix(objectPath, "file://")
	} else if strings.HasPrefix(objectPath, "local://") {
		objectPath = strings.TrimPrefix(objectPath, "local://")
	}
	if ofs, ok := c.fs.(OpenFileFS); ok {
		w, err := ofs.OpenFile(objectPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			return err
		}
		_, err = io.Copy(w, obj)
		return err
	}
	return fmt.Errorf("not support put object")
}

// HeadObject implements storage.Storage. file:// and local:// prefixes are
// stripped from objectPath.
func (c Client) HeadObject(ctx context.Context, objectPath string) (obj *storage.Object, err error) {
	if strings.HasPrefix(objectPath, "file://") {
		objectPath = strings.TrimPrefix(objectPath, "file://")
	} else if strings.HasPrefix(objectPath, "local://") {
		objectPath = strings.TrimPrefix(objectPath, "local://")
	}
	stat, err := os.Stat(objectPath)
	if err != nil {
		return nil, err
	}
	return &storage.Object{
		Key:          objectPath,
		LastModified: stat.ModTime(),
		Size:         stat.Size(),
		StorageClass: c.Type(),
		Metadata:     map[string]string{},
		Mode:         stat.Mode(),
		Headers: map[string][]string{
			"Content-Length": {strconv.FormatInt(stat.Size(), 64)},
			"Last-Modified":  {time.FormatDateTime(stat.ModTime())},
		},
	}, nil
}

// ListObject implements storage.Storage. When recursion is false, only the
// first directory level is walked. file:// and local:// prefixes are stripped.
func (c Client) ListObject(ctx context.Context, objectPrefix string, recursion bool, callback func(key storage.Object)) error {
	logger := log.GetContextLogger(ctx)
	if strings.HasPrefix(objectPrefix, "file://") {
		objectPrefix = strings.TrimPrefix(objectPrefix, "file://")
	} else if strings.HasPrefix(objectPrefix, "local://") {
		objectPrefix = strings.TrimPrefix(objectPrefix, "local://")
	}
	return fs.WalkDir(c.fs, objectPrefix, func(path string, info fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if path == "." {
				return nil
			}
			if !recursion {
				return fs.SkipDir
			}
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			o := storage.Object{
				Key:          path,
				StorageClass: c.Type(),
			}
			fileInfo, err := info.Info()
			if err != nil {
				level.Warn(logger).Log("msg", "failed to get file info", "path", path, "err", err)
			} else {
				o.LastModified = fileInfo.ModTime()
				o.Size = fileInfo.Size()
			}
			callback(o)
		}
		return nil
	})
}

// Type implements storage.Storage. It returns the backend kind such as local or zip.
func (c Client) Type() string {
	return c.fsType
}

// NewClient wraps fs as a storage client. When fs is nil, v["type"] selects
// zip, in-memory/inmemory, or local/file, and v["base"] is the root path or zip file.
func NewClient(_ context.Context, fs fs.FS, v storage.ConfigProvider) (*Client, error) {
	var storageType string
	var base string
	if v != nil {
		storageType = v.GetString("type")
		base = v.GetString("base")
	}
	var err error
	if fs == nil {
		switch storageType {
		case "zip":
			if fs, err = zip.OpenReader(base); err != nil {
				return nil, err
			}
		case "in-memory", "inmemory":
			fs = afero.NewIOFS(afero.NewMemMapFs())
		case "local", "file":
			fs = afero.NewIOFS(afero.NewBasePathFs(afero.NewOsFs(), base))
		}
	}
	if storageType == "" {
		switch vfs := fs.(type) {
		case afero.IOFS:
			storageType = vfs.Name()
		default:
			storageType = "unknown"
		}
	}
	return &Client{
		fs:     fs,
		fsType: storageType,
		o:      Options{Base: base, Type: storageType},
	}, nil
}

func init() {
	storage.RegisterStorage("local", func(ctx context.Context, v storage.ConfigProvider) (storage.Storage, error) {
		return NewClient(ctx, nil, v)
	})
	storage.RegisterStorage("file", func(ctx context.Context, v storage.ConfigProvider) (storage.Storage, error) {
		return NewClient(ctx, nil, v)
	})
	storage.RegisterStorage("zip", func(ctx context.Context, v storage.ConfigProvider) (storage.Storage, error) {
		return NewClient(ctx, nil, v)
	})
	storage.RegisterStorage("in-memory", func(ctx context.Context, v storage.ConfigProvider) (storage.Storage, error) {
		return NewClient(ctx, nil, v)
	})
	storage.RegisterStorage("inmemory", func(ctx context.Context, v storage.ConfigProvider) (storage.Storage, error) {
		return NewClient(ctx, nil, v)
	})
}
