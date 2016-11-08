// Copyright 2016 The appc Authors
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

package log

import (
	"io"
	stdlog "log"
)

// Logger is the interface that enables logging.
// It is compatible with the stdlib "log" methods.
// It is also compatible with https://godoc.org/github.com/Sirupsen/logrus#StdLogger.
type Logger interface {
	Print(...interface{})
	Printf(string, ...interface{})
	Println(...interface{})
}

func NewStdLogger(out io.Writer) Logger {
	return stdlog.New(out, "", 0)
}

type nopLogger struct{}

func NewNopLogger() Logger {
	return &nopLogger{}
}

func (l *nopLogger) Print(...interface{}) {
	// nop
}

func (l *nopLogger) Printf(string, ...interface{}) {
	// nop
}

func (l *nopLogger) Println(...interface{}) {
	// nop
}
