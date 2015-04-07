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
	"encoding/json"
	"fmt"
	"io/ioutil"
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

func TestConfigMerge(t *testing.T) {
	dir, err := ioutil.TempDir("", tstprefix)
	if err != nil {
		panic(fmt.Sprintf("Failed to create temporary directory: %v", err))
	}
	defer os.RemoveAll(dir)
	vendorAuth := filepath.Join("vendor", "auth.d")
	vendorIgnored := filepath.Join(vendorAuth, "ignoreddir")
	customAuth := filepath.Join("custom", "auth.d")
	customIgnored := filepath.Join(customAuth, "ignoreddir")
	dirs := []string{
		"vendor",
		vendorAuth,
		vendorIgnored,
		"custom",
		customAuth,
		customIgnored,
	}
	for _, d := range dirs {
		cd := filepath.Join(dir, d)
		if err := os.Mkdir(cd, 0700); err != nil {
			panic(fmt.Sprintf("Failed to create configuration directory %q: %v", cd, err))
		}
	}
	files := []struct {
		path   string
		domain string
		user   string
		pass   string
	}{
		{filepath.Join(dir, vendorAuth, "endocode.json"), "endocode.com", "vendor_user1", "vendor_password1"},
		{filepath.Join(dir, vendorAuth, "coreos.json"), "coreos.com", "vendor_user2", "vendor_password2"},
		{filepath.Join(dir, vendorAuth, "ignoredfile"), "example1.com", "ignored_user1", "ignored_password1"},
		{filepath.Join(dir, vendorIgnored, "ignoredfile"), "example2.com", "ignored_user2", "ignored_password2"},
		{filepath.Join(dir, vendorIgnored, "ignoredanyway.json"), "example3.com", "ignored_user3", "ignored_password3"},
		{filepath.Join(dir, customAuth, "endocode.json"), "endocode.com", "custom_user1", "custom_password1"},
		{filepath.Join(dir, customAuth, "tectonic.json"), "tectonic.com", "custom_user2", "custom_password2"},
		{filepath.Join(dir, customAuth, "ignoredfile"), "example4.com", "ignored_user4", "ignored_password4"},
		{filepath.Join(dir, customIgnored, "ignoredfile"), "example5.com", "ignored_user5", "ignored_password5"},
		{filepath.Join(dir, customIgnored, "ignoredanyway.json"), "example6.com", "ignored_user6", "ignored_password6"},
	}
	for _, f := range files {
		if err := writeBasicConfig(f.path, f.domain, f.user, f.pass); err != nil {
			panic(fmt.Sprintf("Failed to write configuration file: %v", err))
		}
	}
	cfg, err := GetConfigFrom(filepath.Join(dir, "vendor"), filepath.Join(dir, "custom"))
	if err != nil {
		panic(fmt.Sprintf("Failed to get configuration: %v", err))
	}
	result := make(map[string]http.Header)
	for d, h := range cfg.AuthPerHost {
		result[d] = h.Header()
	}
	expected := map[string]http.Header{
		"endocode.com": http.Header{
			// custom_user1:custom_password1
			authHeader: []string{"Basic Y3VzdG9tX3VzZXIxOmN1c3RvbV9wYXNzd29yZDE="},
		},
		"coreos.com": http.Header{
			// vendor_user2:vendor_password2
			authHeader: []string{"Basic dmVuZG9yX3VzZXIyOnZlbmRvcl9wYXNzd29yZDI="},
		},
		"tectonic.com": http.Header{
			// custom_user2:custom_password2
			authHeader: []string{"Basic Y3VzdG9tX3VzZXIyOmN1c3RvbV9wYXNzd29yZDI="},
		},
	}
	if !reflect.DeepEqual(result, expected) {
		t.Error("Got unexpected results\nResult:\n", result, "\n\nExpected:\n", expected)
	}
}

func writeBasicConfig(path, domain, user, pass string) error {
	type basicv1creds struct {
		User     string `json:"user"`
		Password string `json:"password"`
	}
	type basicv1 struct {
		RktVersion  string       `json:"rktVersion"`
		RktKind     string       `json:"rktKind"`
		Domains     []string     `json:"domains"`
		Type        string       `json:"type"`
		Credentials basicv1creds `json:"credentials"`
	}
	config := &basicv1{
		RktVersion: "v1",
		RktKind:    "auth",
		Domains:    []string{domain},
		Type:       "basic",
		Credentials: basicv1creds{
			User:     user,
			Password: pass,
		},
	}
	raw, err := json.Marshal(config)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(path, raw, 0600)
}
