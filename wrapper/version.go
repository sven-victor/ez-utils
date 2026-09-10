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

package w

import (
	"fmt"
	"time"
)

var (
	// GitCommit is the git commit hash of the build, typically set via -ldflags.
	GitCommit string
	// BuildDate is the build timestamp, typically set via -ldflags.
	BuildDate string
	// GoVersion is the Go toolchain version used to build the binary, typically set via -ldflags.
	GoVersion string
	// Platform is the GOOS/GOARCH of the build, typically set via -ldflags.
	Platform string
	// Version is the semantic version of the build. It defaults to "0.0.0".
	Version string = "0.0.0"
)

var timeFormat = []string{
	time.RFC3339,
	time.RFC3339Nano,
	time.DateTime,
	time.ANSIC,
	time.UnixDate,
	time.RubyDate,
	time.RFC822,
	time.RFC822Z,
	time.RFC850,
	time.RFC1123,
	time.RFC1123Z,
	time.RFC822,
	time.RFC822Z,
	time.RFC850,
	time.RFC1123,
	time.RFC1123Z,
	"2006-01-02T15:04:05Z0700",
}

// AddVersionFlags normalizes BuildDate to RFC3339 when possible and passes the short and full version strings to flagFunc.
func AddVersionFlags(flagFunc func(shortVersion, fullVersion string)) {
	for _, tf := range timeFormat {
		buildDate, err := time.Parse(tf, BuildDate)
		if err == nil {
			BuildDate = buildDate.UTC().Format(time.RFC3339)
			break
		}
	}
	flagFunc(Version, fmt.Sprintf(
		"%s, build %s/%s [%s/%s]\n",
		Version, GitCommit, BuildDate, GoVersion, Platform,
	))
}
