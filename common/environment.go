// Copyright 2016 The rkt Authors
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

package common

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/rkt/rkt/pkg/user"

	"github.com/appc/spec/schema/types"
)

const DefaultPath = "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"

var defaultEnv = map[string]string{
	"PATH":    DefaultPath,
	"SHELL":   "/bin/sh",
	"USER":    "root",
	"LOGNAME": "root",
	"HOME":    "/root",
}

// WriteEnvFile creates an environment file for given app name.  To
// ensure the minimum required environment variables by the appc spec
// are set to sensible defaults, env should be the result of calling
// ComposeEnviron.  The containing directory and its ancestors will be
// created if necessary.
func WriteEnvFile(env []string, uidRange *user.UidRange, envFilePath string) error {
	ef := bytes.Buffer{}

	for _, v := range env {
		fmt.Fprintf(&ef, "%s\n", v)
	}

	if err := os.MkdirAll(filepath.Dir(envFilePath), 0755); err != nil {
		return err
	}

	if err := ioutil.WriteFile(envFilePath, ef.Bytes(), 0644); err != nil {
		return err
	}

	if err := user.ShiftFiles([]string{envFilePath}, uidRange); err != nil {
		return err
	}

	return nil
}

// ReadEnvFileRaw reads the environment file, returning it as a
// slice of strings, each expected but not checked to be of the form
// "key=value".  (The suffix leaves room for a function which parallels
// WriteEnvFile, which splits each string and has a return type of
// types.Environment.)
func ReadEnvFileRaw(envFilePath string) ([]string, error) {
	var env []string
	var envFile *os.File
	var err error
	if envFile, err = os.Open(envFilePath); err != nil {
		return nil, err
	}
	defer envFile.Close()
	scanner := bufio.NewScanner(envFile)
	for scanner.Scan() {
		env = append(env, scanner.Text())
	}
	if err = scanner.Err(); err != nil {
		return nil, err
	}
	return env, nil
}

// ComposeEnviron formats the environment into a slice of strings,
// each of the form "key=value".  The minimum required environment
// variables by the appc spec will be set to sensible defaults here if
// they're not provided by env.
func ComposeEnviron(env types.Environment) []string {
	var composed []string

	for dk, dv := range defaultEnv {
		if _, exists := env.Get(dk); !exists {
			composed = append(composed, fmt.Sprintf("%s=%s", dk, dv))
		}
	}

	for _, e := range env {
		composed = append(composed, fmt.Sprintf("%s=%s", e.Name, e.Value))
	}
	return composed
}
