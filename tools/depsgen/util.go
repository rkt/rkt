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

package main

import (
	"fmt"
	"os"
	"strings"
)

// replacements creates a map from passed strings. This function
// expects an even number of strings, otherwise it will bail out. Odd
// (first, third and so on) strings are keys, even (second, fourth and
// so on) strings are values.
func replacements(kv ...string) map[string]string {
	if len(kv)%2 != 0 {
		fmt.Fprintf(os.Stderr, "Expected even number of strings in replacements\n")
		os.Exit(1)
	}
	r := make(map[string]string, len(kv))
	lastKey := ""
	for i, kv := range kv {
		if i%2 == 0 {
			lastKey = kv
		} else {
			r[lastKey] = kv
		}
	}
	return r
}

// replacePlaceholders replaces placeholders with values in kv in
// initial str. Placeholders are in form of !!!FOO!!!, but those
// passed here should be without exclamation marks.
func replacePlaceholders(str string, kv ...string) string {
	for ph, value := range replacements(kv...) {
		str = strings.Replace(str, "!!!"+ph+"!!!", value, -1)
	}
	return str
}
