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
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/coreos/rkt/Godeps/_workspace/src/github.com/ThomasRooney/gexpect"
	taas "github.com/coreos/rkt/Godeps/_workspace/src/github.com/endocode/test-aci-auth-server/lib"
	"github.com/coreos/rkt/rkt/config"
)

func TestAuthSanity(t *testing.T) {
	skipUnsafe(t)
	removeDataDir(t)
	server := runServer(t, taas.None)
	defer server.Close()
	runRktBang(t, server.URL, "none1")
}

func TestAuthBasic(t *testing.T) {
	skipUnsafe(t)
	removeDataDir(t)
	defer removeAllConfig(t)
	server := runServer(t, taas.Basic)
	defer server.Close()
	basicNoConfig(t, server)
	basicWithCustomConfig(t, server)
	basicWithVendorConfig(t, server)
}

func basicNoConfig(t *testing.T, server *taas.Server) {
	removeAllConfig(t)
	runRkt401(t, server.URL, "basic-no-config")
}

func basicWithCustomConfig(t *testing.T, server *taas.Server) {
	removeAllConfig(t)
	writeCustomConfig(t, "test.json", server.Conf)
	runRktBang(t, server.URL, "basic-custom-config")
}

func basicWithVendorConfig(t *testing.T, server *taas.Server) {
	removeAllConfig(t)
	writeVendorConfig(t, "test.json", server.Conf)
	runRktBang(t, server.URL, "basic-vendor-config")
}

func TestAuthOauth(t *testing.T) {
	skipUnsafe(t)
	removeDataDir(t)
	defer removeAllConfig(t)
	server := runServer(t, taas.Oauth)
	defer server.Close()
	oauthNoConfig(t, server)
	oauthCustomConfig(t, server)
	oauthVendorConfig(t, server)
}

func oauthNoConfig(t *testing.T, server *taas.Server) {
	removeAllConfig(t)
	runRkt401(t, server.URL, "oauth1")
}

func oauthCustomConfig(t *testing.T, server *taas.Server) {
	removeAllConfig(t)
	writeCustomConfig(t, "test.json", server.Conf)
	runRktBang(t, server.URL, "oauth-custom-config")
}

func oauthVendorConfig(t *testing.T, server *taas.Server) {
	removeAllConfig(t)
	writeVendorConfig(t, "test.json", server.Conf)
	runRktBang(t, server.URL, "oauth-vendor-config")
}

func TestAuthOverride(t *testing.T) {
	skipUnsafe(t)
	removeDataDir(t)
	defer removeAllConfig(t)
	server := runServer(t, taas.Oauth)
	defer server.Close()
	validVendorInvalidCustom(t, server)
	invalidVendorValidCustom(t, server)
}

func validVendorInvalidCustom(t *testing.T, server *taas.Server) {
	removeAllConfig(t)
	writeVendorConfig(t, "test.json", server.Conf)
	runRktBang(t, server.URL, "oauth-vvic-1")
	writeCustomConfig(t, "test.json", getInvalidOAuthConfig(server.Conf))
	runRkt401(t, server.URL, "oauth-vvic-2")
}

func invalidVendorValidCustom(t *testing.T, server *taas.Server) {
	removeAllConfig(t)
	writeVendorConfig(t, "test.json", getInvalidOAuthConfig(server.Conf))
	runRkt401(t, server.URL, "oauth-ivvc-1")
	writeCustomConfig(t, "test.json", server.Conf)
	runRktBang(t, server.URL, "oauth-ivvc-2")
}

func TestAuthIgnore(t *testing.T) {
	skipUnsafe(t)
	removeDataDir(t)
	defer removeAllConfig(t)
	server := runServer(t, taas.Oauth)
	defer server.Close()
	bogusFiles(t, server)
	subdirectories(t, server)
}

func bogusFiles(t *testing.T, server *taas.Server) {
	removeAllConfig(t)
	writeVendorConfig(t, "README", "This is vendor config")
	writeCustomConfig(t, "README", "This is custom config")
	writeVendorConfig(t, "test.notjson", server.Conf)
	writeCustomConfig(t, "test.notjson", server.Conf)
	runRkt401(t, server.URL, "oauth-bogus-files")
}

func subdirectories(t *testing.T, server *taas.Server) {
	removeAllConfig(t)
	customSubdir := filepath.Join(config.DefaultCustomPath, "subdir")
	vendorSubdir := filepath.Join(config.DefaultVendorPath, "subdir")
	writeConfig(t, customSubdir, "test.json", server.Conf)
	writeConfig(t, vendorSubdir, "test.json", server.Conf)
	runRkt401(t, server.URL, "oauth-subdirectories")
}

// TODO (krnowak): Remove this when we will be able to specify
// different custom and vendor configuration directories.
const unsafeEnvVar = "RKT_ENABLE_DESTRUCTIVE_TESTS"

func skipUnsafe(t *testing.T) {
	if os.Getenv(unsafeEnvVar) != "1" {
		t.Skipf("%s envvar is not specified or has value different than 1, skipping the test", unsafeEnvVar)
	}
}

func removeDataDir(t *testing.T) {
	if err := os.RemoveAll("/var/lib/rkt"); err != nil {
		t.Fatalf("Failed to remove /var/lib/rkt: %v", err)
	}
}

func runServer(t *testing.T, auth taas.Type) *taas.Server {
	goTool := getAbs(t, "go")
	acTool := getAbs(t, "../bin/actool")
	server, err := taas.NewServerWithPaths(auth, 20, acTool, goTool)
	if err != nil {
		t.Fatalf("Could not start server: %v", err)
	}
	go serverHandler(t, server)
	return server
}

func getAbs(t *testing.T, tool string) string {
	baseTool := filepath.Base(tool)
	pathTool, err := exec.LookPath(tool)
	if err != nil {
		t.Fatalf("Could not find %s, required to run server: %v", baseTool, err)
	}
	absTool, err := filepath.Abs(pathTool)
	if err != nil {
		t.Fatalf("Could not get absolute path of %s, required to run server: %v", baseTool, err)
	}
	return absTool
}

func serverHandler(t *testing.T, server *taas.Server) {
	for {
		select {
		case _, ok := <-server.Stop:
			if ok {
				t.Log("Closing server")
				server.Close()
			}
			return
		case msg, ok := <-server.Msg:
			if ok {
				t.Logf("server: %v", msg)
			}
		}
	}
}

func runRktBang(t *testing.T, host, dir string) {
	child := runRkt(t, host, dir)
	defer child.Wait()
	expectBang(t, child)
}

func runRkt401(t *testing.T, host, dir string) {
	child := runRkt(t, host, dir)
	defer child.Wait()
	expect401(t, child)
}

// TODO (krnowak): Use --dir option when we also add
// --vendor-config-dir and --custom-config-dir options.
func runRkt(t *testing.T, host, dir string) *gexpect.ExpectSubprocess {
	cmd := rktCmd(fmt.Sprintf(`run %s/%s/prog.aci`, host, dir))
	t.Logf("Running rkt: %s", cmd)
	child, err := gexpect.Spawn(cmd)
	if err != nil {
		t.Fatalf("Failed to run rkt: %v", err)
	}
	return child
}

func rktCmd(rest string) string {
	return fmt.Sprintf(`../bin/rkt --debug --insecure-skip-verify %s`, rest)
}

func expectBang(t *testing.T, child *gexpect.ExpectSubprocess) {
	expectLine(t, child, "BANG!")
}

func expect401(t *testing.T, child *gexpect.ExpectSubprocess) {
	expectLine(t, child, "error downloading ACI: bad HTTP status code: 401")
}

func expectLine(t *testing.T, child *gexpect.ExpectSubprocess, line string) {
	if err := child.Expect(line); err != nil {
		t.Fatalf("Didn't receive expected output %q", line)
	}
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
