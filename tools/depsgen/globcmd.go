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
	"path/filepath"
	"strings"
)

const (
	// globMakeFunction is a template for generating all files for
	// given set of wildcards. See globMakeWildcard.
	globMakeFunction = "$(shell stat --format \"%n: %F\" !!!WILDCARDS!!! | grep -e 'regular file$$' | cut -f1 -d:)"
	// globMakeWildcard is a template for call wildcard function
	// for in a given directory with a given suffix. First
	// wildcard is for normal files, second wildcard is for files
	// beginning with a dot, which are normally not taken into
	// account by wildcard.
	globMakeWildcard = "$(wildcard !!!DIR!!!/*!!!SUFFIX!!!) $(wildcard !!!DIR!!!/.*!!!SUFFIX!!!)"
	globCmd          = "glob"
)

func init() {
	cmds[globCmd] = globDeps
}

func globDeps(args []string) string {
	target, suffix, files := getGlobArgs(args)
	return GenerateFileDeps(target, getGlobMakeFunction(files, suffix), files)
}

// getGlobArgs parses given parameters and returns a target, a suffix
// and a list of files.
func getGlobArgs(args []string) (string, string, []string) {
	f, target := standardFlags(globCmd)
	suffix := f.String("suffix", "", "File suffix (example: .go)")

	f.Parse(args)
	if *target == "" {
		fmt.Fprintf(os.Stderr, "--target parameter must be specified and cannot be empty\n")
		os.Exit(1)
	}
	return *target, *suffix, f.Args()
}

// getGlobMakeFunction returns a make snippet which calls wildcard
// function in all directories where given files are and with a given
// suffix.
func getGlobMakeFunction(files []string, suffix string) string {
	dirs := map[string]struct{}{}
	for _, file := range files {
		dirs[filepath.Dir(file)] = struct{}{}
	}
	makeWildcards := make([]string, 0, len(dirs))
	for dir := range dirs {
		str := replacePlaceholders(globMakeWildcard, "SUFFIX", suffix, "DIR", dir)
		makeWildcards = append(makeWildcards, str)
	}
	return replacePlaceholders(globMakeFunction, "WILDCARDS", strings.Join(makeWildcards, " "))
}
