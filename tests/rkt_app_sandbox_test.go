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

// Disabled on kvm due to https://github.com/coreos/rkt/issues/3382
// +build !fly,!kvm

package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/coreos/rkt/tests/testutils"
)

// TestAppSandboxSmoke is a basic smoke test for `rkt app` sandbox
// and related commands.
func TestAppSandboxSmoke(t *testing.T) {
	actionTimeout := 30 * time.Second
	imageName := "coreos.com/rkt-inspect/hello"
	appName := "hello-app"
	msg := "HelloFromAppInSandbox"

	aciHello := patchTestACI("rkt-inspect-hello.aci", "--name="+imageName, "--exec=/inspect --print-msg="+msg)
	defer os.Remove(aciHello)

	ctx := testutils.NewRktRunCtx()
	defer ctx.Cleanup()

	if err := os.Setenv("RKT_EXPERIMENT_APP", "true"); err != nil {
		panic(err)
	}
	defer os.Unsetenv("RKT_EXPERIMENT_APP")

	fetch := ctx.ExecCmd("fetch", "--insecure-options=image", aciHello)
	fetch.Env = append(fetch.Env, "RKT_EXPERIMENT_APP=true")
	t.Log("Running", fetch.Args)
	if out, err := fetch.CombinedOutput(); err != nil {
		t.Fatal(err, "output", out)
	}

	tmpDir := createTempDirOrPanic("rkt-test-cri-")
	uuidFile := filepath.Join(tmpDir, "uuid")
	defer os.RemoveAll(tmpDir)

	rkt := ctx.Cmd() + " app sandbox --uuid-file-save=" + uuidFile
	child := spawnOrFail(t, rkt)

	// wait for the sandbox to start
	podUUID, err := waitPodReady(ctx, t, uuidFile, actionTimeout)
	if err != nil {
		t.Fatal(err)
	}

	add := ctx.ExecCmd("app", "add", "--debug", podUUID, imageName, "--name="+appName)
	t.Log("Running", add.Args)
	if out, err := add.CombinedOutput(); err != nil {
		t.Fatal(err, "output", out)
	}

	start := ctx.ExecCmd("app", "start", "--debug", podUUID, "--app="+appName)
	t.Log("Running", start.Args)
	if out, err := start.CombinedOutput(); err != nil {
		t.Fatal(err, "output", out)
	}

	if err := expectTimeoutWithOutput(child, msg, actionTimeout); err != nil {
		t.Fatalf("Expected %q but not found: %v", msg, err)
	}

	remove := ctx.ExecCmd("app", "rm", "--debug", podUUID, "--app="+appName)
	t.Log("Running", remove.Args)
	if out, err := remove.CombinedOutput(); err != nil {
		t.Fatal(err, "output", out)
	}

	stop := ctx.ExecCmd("stop", podUUID)
	t.Log("Running", stop.Args)
	if out, err := stop.CombinedOutput(); err != nil {
		t.Fatal(err, "output", out)
	}

	waitOrFail(t, child, 0)
}
