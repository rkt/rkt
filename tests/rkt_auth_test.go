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

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/coreos/rkt/Godeps/_workspace/src/github.com/ThomasRooney/gexpect"
	"github.com/coreos/rkt/rkt/config"
	taas "github.com/coreos/rkt/tests/test-auth-server/aci"
)

func TestAuthSanity(t *testing.T) {
	skipDestructive(t)
	removeDataDir(t)
	server := runServer(t, taas.None)
	defer server.Close()
	successfulRunRkt(t, server.URL, "sanity")
}

const (
	authSuccessfulDownload = "Authentication succeeded."
	authFailedDownload     = "error downloading ACI: bad HTTP status code: 401"
)

type genericAuthTest struct {
	name          string
	useServerConf bool
	confDir       string
	expectedLine  string
}

func TestAuthBasic(t *testing.T) {
	tests := []genericAuthTest{
		{"basic-no-config", false, "", authFailedDownload},
		{"basic-custom-config", true, config.DefaultCustomPath, authSuccessfulDownload},
		{"basic-vendor-config", true, config.DefaultVendorPath, authSuccessfulDownload},
	}
	testAuthGeneric(t, taas.Basic, tests)
}

func TestAuthOauth(t *testing.T) {
	tests := []genericAuthTest{
		{"oauth-no-config", false, "", authFailedDownload},
		{"oauth-custom-config", true, config.DefaultCustomPath, authSuccessfulDownload},
		{"oauth-vendor-config", true, config.DefaultVendorPath, authSuccessfulDownload},
	}
	testAuthGeneric(t, taas.Oauth, tests)
}

func testAuthGeneric(t *testing.T, auth taas.Type, tests []genericAuthTest) {
	skipDestructive(t)
	removeDataDir(t)
	defer removeAllConfig(t)
	server := runServer(t, auth)
	defer server.Close()
	for _, tt := range tests {
		removeAllConfig(t)
		if tt.useServerConf {
			writeConfig(t, tt.confDir, "test.json", server.Conf)
		}
		expectedRunRkt(t, server.URL, tt.name, tt.expectedLine)
	}
}

func TestAuthOverride(t *testing.T) {
	skipDestructive(t)
	removeDataDir(t)
	defer removeAllConfig(t)
	server := runServer(t, taas.Oauth)
	defer server.Close()
	tests := []struct {
		vendorConfig         string
		customConfig         string
		name                 string
		resultBeforeOverride string
		resultAfterOverride  string
	}{
		{server.Conf, getInvalidOAuthConfig(server.Conf), "valid-vendor-invalid-custom", authSuccessfulDownload, authFailedDownload},
		{getInvalidOAuthConfig(server.Conf), server.Conf, "invalid-vendor-valid-custom", authFailedDownload, authSuccessfulDownload},
	}
	for _, tt := range tests {
		removeAllConfig(t)
		writeVendorConfig(t, "test.json", tt.vendorConfig)
		expectedRunRkt(t, server.URL, tt.name+"-1", tt.resultBeforeOverride)
		writeCustomConfig(t, "test.json", tt.customConfig)
		expectedRunRkt(t, server.URL, tt.name+"-2", tt.resultAfterOverride)
	}
}

func TestAuthIgnore(t *testing.T) {
	skipDestructive(t)
	removeDataDir(t)
	defer removeAllConfig(t)
	server := runServer(t, taas.Oauth)
	defer server.Close()
	testAuthIgnoreBogusFiles(t, server)
	testAuthIgnoreSubdirectories(t, server)
}

func testAuthIgnoreBogusFiles(t *testing.T, server *taas.Server) {
	removeAllConfig(t)
	writeVendorConfig(t, "README", "This is vendor config")
	writeCustomConfig(t, "README", "This is custom config")
	writeVendorConfig(t, "test.notjson", server.Conf)
	writeCustomConfig(t, "test.notjson", server.Conf)
	failedRunRkt(t, server.URL, "oauth-bogus-files")
}

func testAuthIgnoreSubdirectories(t *testing.T, server *taas.Server) {
	removeAllConfig(t)
	customSubdir := filepath.Join(config.DefaultCustomPath, "subdir")
	vendorSubdir := filepath.Join(config.DefaultVendorPath, "subdir")
	writeConfig(t, customSubdir, "test.json", server.Conf)
	writeConfig(t, vendorSubdir, "test.json", server.Conf)
	failedRunRkt(t, server.URL, "oauth-subdirectories")
}

func runServer(t *testing.T, auth taas.Type) *taas.Server {
	server, err := taas.NewServerWithPaths(auth, 20, "../bin/actool", "go")
	if err != nil {
		t.Fatalf("Could not start server: %v", err)
	}
	go serverHandler(t, server)
	return server
}

func serverHandler(t *testing.T, server *taas.Server) {
	for {
		select {
		case msg, ok := <-server.Msg:
			if ok {
				t.Logf("server: %v", msg)
			}
		}
	}
}

func successfulRunRkt(t *testing.T, host, dir string) {
	expectedRunRkt(t, host, dir, authSuccessfulDownload)
}

func failedRunRkt(t *testing.T, host, dir string) {
	expectedRunRkt(t, host, dir, authFailedDownload)
}

func expectedRunRkt(t *testing.T, host, dir, line string) {
	child := runRkt(t, host, dir)
	defer child.Wait()
	if err := child.Expect(line); err != nil {
		t.Fatalf("Didn't receive expected output %q", line)
	}
}

// TODO (krnowak): Use --dir option when we also add
// --vendor-config-dir and --custom-config-dir options. Then we can
// remove destructive tests checks.

// runRkt tries to fetch and run a prog.aci from host within given
// directory on host. Note that directory can be anything - it's
// useful for ensuring that image name is unique and for descriptive
// purposes.
func runRkt(t *testing.T, host, dir string) *gexpect.ExpectSubprocess {
	cmd := fmt.Sprintf(`../bin/rkt --debug --insecure-skip-verify run %s/%s/prog.aci`, host, dir)
	t.Logf("Running rkt: %s", cmd)
	child, err := gexpect.Spawn(cmd)
	if err != nil {
		t.Fatalf("Failed to run rkt: %v", err)
	}
	return child
}

func removeAllConfig(t *testing.T) {
	dirs := []string{
		authDir(config.DefaultCustomPath),
		authDir(config.DefaultVendorPath),
	}
	for _, p := range dirs {
		if err := os.RemoveAll(p); err != nil {
			t.Fatalf("Failed to remove config directory %q: %v", p, err)
		}
		if err := os.MkdirAll(p, 0755); err != nil {
			t.Fatalf("Failed to create config directory %q: %v", p, err)
		}
	}
}

func writeCustomConfig(t *testing.T, filename, contents string) {
	writeConfig(t, config.DefaultCustomPath, filename, contents)
}

func writeVendorConfig(t *testing.T, filename, contents string) {
	writeConfig(t, config.DefaultVendorPath, filename, contents)
}

func writeConfig(t *testing.T, baseDir, filename, contents string) {
	dir := authDir(baseDir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("Failed to create config directory %q: %v", dir, err)
	}
	path := filepath.Join(dir, filename)
	os.Remove(path)
	if err := ioutil.WriteFile(path, []byte(contents), 0644); err != nil {
		t.Fatalf("Failed to write file %q: %v", path, err)
	}
}

func authDir(confDir string) string {
	return filepath.Join(confDir, "auth.d")
}

func getInvalidOAuthConfig(conf string) string {
	return strings.Replace(conf, "sometoken", "someobviouslywrongtoken", 1)
}
