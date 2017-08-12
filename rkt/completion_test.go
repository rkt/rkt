// Copyright 2017 The rkt Authors
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

package main

import (
	"bytes"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/rkt/rkt/pkg/log"
)

func TestBashCompletion(t *testing.T) {
	tests := []struct {
		ExitCode int
		Args     []string
	}{
		// Command expects at least one parameter.
		{254, []string{}},
		// Only single argument is expected.
		{254, []string{"two", "args"}},
		// Fish shell is not supported.
		{254, []string{"fish"}},
		// Bash completion should succeed.
		{0, []string{"bash"}},
	}

	var buf bytes.Buffer
	handler := newCompletion(&buf)
	stderr = log.New(ioutil.Discard, "", false)

	for ii, tt := range tests {
		buf.Reset()
		code := handler(cmdCompletion, tt.Args)
		if code != tt.ExitCode {
			t.Errorf("#%d: got %v exit code, want %v", ii, code, tt.ExitCode)
		}
	}

	// Check that generated output contains custom bash functions.
	buf.Reset()
	handler(cmdCompletion, []string{"bash"})

	output := buf.String()
	if !strings.Contains(output, bashCompletionFunc) {
		t.Errorf("it is expected custom bash functions in the output")
	}
}
