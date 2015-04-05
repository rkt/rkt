// Copyright 2015 CoreOS, Inc.
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

package config

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

const tstprefix = "config-test"

func tmpConfigFile(prefix string) (*os.File, error) {
	dir := os.TempDir()
	idx := 0
	tries := 10000
	for i := 0; i < tries; i++ {
		name := filepath.Join(dir, fmt.Sprintf("%s%d.json", prefix, idx))
		f, err := os.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0600)
		if os.IsExist(err) {
			idx++
			continue
		}
		return f, err
	}
	return nil, fmt.Errorf("Failed to get tmpfile after %d tries", tries)
}

func TestConfig(t *testing.T) {
	tests := []struct {
		contents string
		expected map[string]http.Header
		fail     bool
	}{
		{"bogus contents", nil, true},
		{`{"bogus": {"foo": "bar"}}`, nil, true},
		{`{"rktKind": "foo"}`, nil, true},
		{`{"rktKind": "auth", "rktVersion": "foo"}`, nil, true},
		{`{"rktKind": "auth", "rktVersion": "v1"}`, nil, true},
		{`{"rktKind": "auth", "rktVersion": "v1", "domains": "foo"}`, nil, true},
		{`{"rktKind": "auth", "rktVersion": "v1", "domains": []}`, nil, true},
		{`{"rktKind": "auth", "rktVersion": "v1", "domains": ["coreos.com"]}`, nil, true},
		{`{"rktKind": "auth", "rktVersion": "v1", "domains": ["coreos.com"], "type": "foo"}`, nil, true},
		{`{"rktKind": "auth", "rktVersion": "v1", "domains": ["coreos.com"], "type": "basic"}`, nil, true},
		{`{"rktKind": "auth", "rktVersion": "v1", "domains": ["coreos.com"], "type": "basic", "credentials": {}}`, nil, true},
		{`{"rktKind": "auth", "rktVersion": "v1", "domains": ["coreos.com"], "type": "basic", "credentials": {"user": ""}}`, nil, true},
		{`{"rktKind": "auth", "rktVersion": "v1", "domains": ["coreos.com"], "type": "basic", "credentials": {"user": "bar"}}`, nil, true},
		{`{"rktKind": "auth", "rktVersion": "v1", "domains": ["coreos.com"], "type": "basic", "credentials": {"user": "bar", "password": ""}}`, nil, true},
		{`{"rktKind": "auth", "rktVersion": "v1", "domains": ["coreos.com"], "type": "basic", "credentials": {"user": "bar", "password": "baz"}}`, map[string]http.Header{"coreos.com": {"Authorization": []string{"Basic YmFyOmJheg=="}}}, false},
		{`{"rktKind": "auth", "rktVersion": "v1", "domains": ["coreos.com"], "type": "oauth"}`, nil, true},
		{`{"rktKind": "auth", "rktVersion": "v1", "domains": ["coreos.com"], "type": "oauth", "credentials": {}}`, nil, true},
		{`{"rktKind": "auth", "rktVersion": "v1", "domains": ["coreos.com"], "type": "oauth", "credentials": {"token": ""}}`, nil, true},
		{`{"rktKind": "auth", "rktVersion": "v1", "domains": ["coreos.com"], "type": "oauth", "credentials": {"token": "sometoken"}}`, map[string]http.Header{"coreos.com": {"Authorization": []string{"Bearer sometoken"}}}, false},
	}
	for _, tt := range tests {
		f, err := tmpConfigFile(tstprefix)
		if err != nil {
			panic(fmt.Sprintf("Failed to create tmp config file: %v", err))
		}
		defer f.Close()
		if _, err := f.Write([]byte(tt.contents)); err != nil {
			panic(fmt.Sprintf("Writing config to file failed: %v", err))
		}
		fi, err := f.Stat()
		if err != nil {
			panic(fmt.Sprintf("Stating a tmp config file failed: %v", err))
		}
		cfg := newConfig()
		if err := readFile(cfg, fi, f.Name(), []string{"auth"}); err != nil {
			if !tt.fail {
				t.Errorf("Expected test to succeed, failed unexpectedly (contents: `%s`)", tt.contents)
			}
		} else if tt.fail {
			t.Errorf("Expected test to fail, succeeded unexpectedly (contents: `%s`)", tt.contents)
		} else {
			result := make(map[string]http.Header)
			for k, v := range cfg.AuthPerHost {
				result[k] = v.Header()
			}
			if !reflect.DeepEqual(result, tt.expected) {
				t.Error("Got unexpected results\nResult:\n", result, "\n\nExpected:\n", tt.expected)
			}
		}
	}
}
