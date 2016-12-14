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
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

	aciHello := patchTestACI("rkt-inspect-hello.aci", fmt.Sprintf("--name=%s", imageName), fmt.Sprintf("--exec=/inspect --print-msg=%s", msg))
	defer os.Remove(aciHello)

	ctx := testutils.NewRktRunCtx()
	defer ctx.Cleanup()

	cmd := strings.Fields(fmt.Sprintf("%s fetch --insecure-options=image %s", ctx.Cmd(), aciHello))
	fetchCmd := exec.Command(cmd[0], cmd[1:]...)
	fetchCmd.Env = append(fetchCmd.Env, "RKT_EXPERIMENT_APP=true")
	fetchOutput, err := fetchCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Unexpected error: %v\n%s", err, fetchOutput)
	}

	tmpDir := createTempDirOrPanic("rkt-test-cri-")
	uuidFile := filepath.Join(tmpDir, "uuid")
	defer os.RemoveAll(tmpDir)

	rktCmd := fmt.Sprintf("%s app sandbox --uuid-file-save=%s", ctx.Cmd(), uuidFile)
	err = os.Setenv("RKT_EXPERIMENT_APP", "true")
	if err != nil {
		panic(err)
	}
	child := spawnOrFail(t, rktCmd)
	err = os.Unsetenv("RKT_EXPERIMENT_APP")
	if err != nil {
		panic(err)
	}

	// wait for the sandbox to start
	podUUID, err := waitPodReady(ctx, t, uuidFile, actionTimeout)
	if err != nil {
		t.Fatal(err)
	}

	cmd = strings.Fields(fmt.Sprintf("%s app add --debug %s %s --name=%s", ctx.Cmd(), podUUID, imageName, appName))
	addCmd := exec.Command(cmd[0], cmd[1:]...)
	addCmd.Env = append(addCmd.Env, "RKT_EXPERIMENT_APP=true")
	t.Logf("Running command: %v\n", cmd)
	output, err := addCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Unexpected error: %v\n%s", err, output)
	}

	cmd = strings.Fields(fmt.Sprintf("%s app start --debug %s --app=%s", ctx.Cmd(), podUUID, appName))
	startCmd := exec.Command(cmd[0], cmd[1:]...)
	startCmd.Env = append(startCmd.Env, "RKT_EXPERIMENT_APP=true")
	t.Logf("Running command: %v\n", cmd)
	output, err = startCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Unexpected error: %v\n%s", err, output)
	}

	if err := expectTimeoutWithOutput(child, msg, actionTimeout); err != nil {
		t.Fatalf("Expected %q but not found: %v", msg, err)
	}

	cmd = strings.Fields(fmt.Sprintf("%s app rm --debug %s --app=%s", ctx.Cmd(), podUUID, appName))
	removeCmd := exec.Command(cmd[0], cmd[1:]...)
	removeCmd.Env = append(removeCmd.Env, "RKT_EXPERIMENT_APP=true")
	t.Logf("Running command: %v\n", cmd)
	output, err = removeCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Unexpected error: %v\n%s", err, output)
	}

	cmd = strings.Fields(fmt.Sprintf("%s stop %s", ctx.Cmd(), podUUID))
	t.Logf("Running command: %v\n", cmd)
	output, err = exec.Command(cmd[0], cmd[1:]...).CombinedOutput()
	if err != nil {
		t.Fatalf("Unexpected error: %v\n%s", err, output)
	}

	waitOrFail(t, child, 0)
}
