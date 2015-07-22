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
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
)

const (
	// separator is used to separate tuple elements in go list
	// format string.
	separator = "!_##_!"
	// goMakeFunction is a template for generating all files for a
	// given module in a given repo.
	goMakeFunction = "$(shell $(GO_ENV) \"$(DEPSGENTOOL)\" go --repo \"!!!REPO!!!\" --module \"!!!MODULE!!!\" --mode files)"
)

type depsMode int

const (
	makeMode depsMode = iota
	filesMode
)

func init() {
	cmds["go"] = goDeps
}

func goDeps(args []string) string {
	target, repo, module, mode := getGoArgs(args)
	deps := getPackageDeps(repo, module)
	var result string
	switch mode {
	case makeMode:
		result = Generate(target, getGoMakeFunction(repo, module), deps)
	case filesMode:
		result = strings.Join(deps, " ")
	default:
		fmt.Fprintf(os.Stderr, "Wrong mode, shouldn't happen.\n")
		os.Exit(1)
	}
	return result
}

// getGoMakeFunction returns a make snippet which will call depsgen go
// with "files" mode.
func getGoMakeFunction(repo, module string) string {
	return replacePlaceholders(goMakeFunction, "REPO", repo, "MODULE", module)
}

// getArgs parses given parameters and returns target, repo, module and
// mode. If mode is "files", then target is optional.
func getGoArgs(args []string) (string, string, string, depsMode) {
	f := flag.NewFlagSet("depsgen go", flag.ExitOnError)
	target := f.String("target", "", "Make target (example: $(FOO_BINARY))")
	repo := f.String("repo", "", "Go repo (example: github.com/coreos/rkt)")
	module := f.String("module", "", "Module inside Go repo (example: stage1)")
	mode := f.String("mode", "make", "Mode to use (make - print deps as makefile [default], files - print a list of files)")

	f.Parse(args)
	if *repo == "" {
		fmt.Fprintf(os.Stderr, "--repo parameter must be specified and cannot be empty\n")
		os.Exit(1)
	}
	if *module == "" {
		fmt.Fprintf(os.Stderr, "--module parameter must be specified and cannot be empty\n")
		os.Exit(1)
	}

	var dMode depsMode

	switch *mode {
	case "make":
		dMode = makeMode
		if *target == "" {
			fmt.Fprintf(os.Stderr, "--target parameter must be specified and cannot be empty when using 'make' mode\n")
			os.Exit(1)
		}
	case "files":
		dMode = filesMode
	default:
		fmt.Fprintf(os.Stderr, "unknown --mode parameter '%s' - expected either 'make' or 'files'\n", *mode)
		os.Exit(1)
	}
	return *target, *repo, *module, dMode
}

// getPackageDeps returns a list of files that are used to build a
// module in a given repo.
func getPackageDeps(repo, module string) []string {
	pkg := path.Join(repo, module)
	deps := []string{pkg}
	for _, d := range getDeps(pkg) {
		if strings.HasPrefix(d, repo) {
			deps = append(deps, d)
		}
	}
	return getFiles(repo, deps)
}

// getDeps gets all dependencies, direct or indirect, of a given
// package.
func getDeps(pkg string) []string {
	rawDeps := run(goList([]string{"Deps"}, []string{pkg}))
	// we expect only one line
	if len(rawDeps) != 1 {
		return []string{}
	}
	return sliceRawSlice(rawDeps[0])
}

// getFiles returns a list of files that are in given packages. File
// paths are "relative" to passed repo.
func getFiles(repo string, pkgs []string) []string {
	params := []string{
		"ImportPath",
		"GoFiles",
		"CgoFiles",
	}
	allFiles := []string{}
	rawTuples := run(goList(params, pkgs))
	for _, raw := range rawTuples {
		tuple := sliceRawTuple(raw)
		module := strings.TrimPrefix(tuple[0], repo+"/")
		files := append(sliceRawSlice(tuple[1]), sliceRawSlice(tuple[2])...)
		for i := 0; i < len(files); i++ {
			files[i] = filepath.Join(module, files[i])
		}
		allFiles = append(allFiles, files...)
	}
	return allFiles
}

// goList returns an array of strings describing go list invocation
// with format string consisting all given params separated with
// !_##_! for all given packages.
func goList(params, pkgs []string) []string {
	templateParams := make([]string, 0, len(params))
	for _, p := range params {
		templateParams = append(templateParams, "{{."+p+"}}")
	}
	return append([]string{
		"go",
		"list",
		"-f", strings.Join(templateParams, separator),
	}, pkgs...)
}

// run executes given argument list and captures its output. The
// output is sliced into lines with empty lines being discarded.
func run(argv []string) []string {
	cmd := exec.Command(argv[0], argv[1:]...)
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running %s: %v: %s", strings.Join(argv, " "), err, stderr.String())
		os.Exit(1)
	}
	rawLines := strings.Split(stdout.String(), "\n")
	lines := make([]string, 0, len(rawLines))
	for _, line := range rawLines {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			lines = append(lines, trimmed)
		}
	}
	return lines
}

// sliceRawSlice slices given string representation of a slice into
// slice of strings.
func sliceRawSlice(s string) []string {
	s = strings.TrimPrefix(s, "[")
	s = strings.TrimSuffix(s, "]")
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	a := strings.Split(s, " ")
	return a
}

// sliceRawTuple slices given string along !_##_! separator to slice
// of strings. Returned slice might need another round of slicing with
// sliceRawSlice.
func sliceRawTuple(t string) []string {
	return strings.Split(t, separator)
}
