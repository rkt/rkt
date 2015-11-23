// Copyright 2015 The rkt Authors
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

package image

import (
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/coreos/rkt/Godeps/_workspace/src/github.com/coreos/ioprogress"
)

// writeSyncer is an interface that wraps io.Writer and a Sync method.
type writeSyncer interface {
	io.Writer
	Sync() error
}

// getIoProgressReader returns a reader that wraps the HTTP response
// body, so it prints a pretty progress bar when reading data from it.
func getIoProgressReader(label string, res *http.Response) io.Reader {
	prefix := "Downloading " + label
	fmtBytesSize := 18
	barSize := int64(80 - len(prefix) - fmtBytesSize)
	bar := ioprogress.DrawTextFormatBarForW(barSize, os.Stderr)
	fmtfunc := func(progress, total int64) string {
		// Content-Length is set to -1 when unknown.
		if total == -1 {
			return fmt.Sprintf(
				"%s: %v of an unknown total size",
				prefix,
				ioprogress.ByteUnitStr(progress),
			)
		}
		return fmt.Sprintf(
			"%s: %s %s",
			prefix,
			bar(progress, total),
			ioprogress.DrawTextFormatBytes(progress, total),
		)
	}
	return &ioprogress.Reader{
		Reader:       res.Body,
		Size:         res.ContentLength,
		DrawFunc:     ioprogress.DrawTerminalf(os.Stderr, fmtfunc),
		DrawInterval: time.Second,
	}
}
