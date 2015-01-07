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

package main

import (
	"testing"
)

func TestQuoteExec(t *testing.T) {
	tests := []struct {
		input  []string
		output string
	}{
		{
			input:  []string{`path`, `"arg1"`, `"'arg2'"`, `'arg3'`},
			output: `path "\"arg1\"" "\"\'arg2\'\"" "\'arg3\'"`,
		}, {
			input:  []string{`path`},
			output: `path`,
		}, {
			input:  []string{`path`, ``, `arg2`},
			output: `path "" "arg2"`,
		}, {
			input:  []string{`path`, `"foo\bar"`, `\`},
			output: `path "\"foo\\bar\"" "\\"`,
		},
	}

	for i, tt := range tests {
		o := quoteExec(tt.input)
		if o != tt.output {
			t.Errorf("#%d: expected `%v` got `%v`", i, tt.output, o)
		}
	}
}
