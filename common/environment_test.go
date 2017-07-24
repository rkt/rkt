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

package common

import (
	"fmt"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/appc/spec/schema/types"
	"github.com/rkt/rkt/pkg/user"
)

func addEnvToMap(env types.Environment, m map[string]string) {
	for _, e := range env {
		m[e.Name] = e.Value
	}
}

func addRawToMap(raw []string, m map[string]string) error {
	for _, kv := range raw {
		fields := strings.SplitN(kv, "=", 2)
		if len(fields) != 2 {
			return fmt.Errorf("invalid environment string `%s'", kv)
		}
		m[fields[0]] = fields[1]
	}
	return nil
}

// add m1 to the map m
func addMapToMap(m1 map[string]string, m map[string]string) error {
	for k, v := range m1 {
		m[k] = v
	}
	return nil
}

func checkEnvMap(expected, actual map[string]string) error {
	// check they're equal
	for k0, v0 := range expected {
		v1, ok := actual[k0]
		if !ok {
			return fmt.Errorf("missing environment string for `%s'", k0)
		}
		if v1 != v0 {
			return fmt.Errorf("environment variable `%s' expected `%s', actual `%s'", k0, v0, v1)
		}
	}
	for k1, v1 := range actual {
		_, ok := expected[k1]
		if !ok {
			return fmt.Errorf("unexpected bonus environment variable `%s=%s'", k1, v1)
		}
	}
	return nil
}

func checkEnv(initial types.Environment, actualRaw []string) error {
	// determine what we expect
	expected := make(map[string]string)
	for k, v := range defaultEnv {
		expected[k] = v
	}
	addEnvToMap(initial, expected)

	// determine what we got
	actual := make(map[string]string)
	err := addRawToMap(actualRaw, actual)
	if err != nil {
		return err
	}

	return checkEnvMap(expected, actual)
}

func writeAndReadEnvFile(env types.Environment) ([]string, error) {
	tmpFile := path.Join("/tmp", fmt.Sprintf("rkt-common-environment_test%d", os.Getpid()))
	defer os.Remove(tmpFile)
	err := WriteEnvFile(ComposeEnviron(env), user.NewBlankUidRange(), tmpFile)
	if err != nil {
		return nil, err
	}
	return ReadEnvFileRaw(tmpFile)
}

func TestWriteEnvFile(t *testing.T) {
	tests := []types.Environment{
		{},
		{
			types.EnvironmentVariable{
				Name:  "PATH",
				Value: "/something/stupid/that/will/be/preserved",
			},
			types.EnvironmentVariable{
				Name:  "USER",
				Value: "511",
			},
		},
		{
			types.EnvironmentVariable{
				Name:  "FIRST_NAME",
				Value: "Zaphod",
			},
		},
	}

	for i, tt := range tests {

		result, err := writeAndReadEnvFile(tt)
		if err == nil {
			err = checkEnv(tt, result)
		}
		if err != nil {
			t.Errorf("#%d: %v", i, err)
		}
	}
}

type envAndExpected struct {
	env      types.Environment
	expected []string
}

func checkComposedEnv(expectedRaw, actualRaw []string) error {
	expected := make(map[string]string)

	// these defaults will get overridden by any actual values
	addMapToMap(defaultEnv, expected)

	err := addRawToMap(expectedRaw, expected)
	if err != nil {
		return err
	}
	actual := make(map[string]string)
	err = addRawToMap(actualRaw, actual)
	if err != nil {
		return err
	}
	return checkEnvMap(expected, actual)
}

func TestComposeEnv(t *testing.T) {
	tests := []envAndExpected{
		{
			env:      types.Environment{},
			expected: nil,
		},
		{
			env: types.Environment{
				types.EnvironmentVariable{
					Name:  "PATH",
					Value: "/something/stupid/that/will/be/preserved",
				},
				types.EnvironmentVariable{
					Name:  "USER",
					Value: "511",
				},
			},
			expected: []string{
				"PATH=/something/stupid/that/will/be/preserved",
				"USER=511",
			},
		},
		{
			env: types.Environment{
				types.EnvironmentVariable{
					Name:  "FIRST_NAME",
					Value: "Zaphod",
				},
			},
			expected: []string{
				"FIRST_NAME=Zaphod",
			},
		},
	}

	for i, tt := range tests {

		result := ComposeEnviron(tt.env)
		err := checkComposedEnv(tt.expected, result)
		if err != nil {
			t.Errorf("#%d: %v", i, err)
		}
	}
}
