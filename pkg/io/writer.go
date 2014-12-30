// Copyright 2014 CoreOS, Inc.
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

package io

import "io"

// LimitedWriter is similar to io.LimitedReader; it writes to W but
// limits the amount of data written to just N bytes. Each subsequent call to
// Write() will return a nil error and a count of 0.
type LimitedWriter struct {
	W io.Writer
	N int64
}

func (l *LimitedWriter) Write(data []byte) (n int, err error) {
	if l.N <= 0 {
		return 0, nil
	}
	if int64(len(data)) > l.N {
		data = data[0:l.N]
	}
	n, err = l.W.Write(data)
	l.N -= int64(n)
	return
}
