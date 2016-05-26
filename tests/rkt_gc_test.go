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

// +build host coreos src kvm

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/coreos/rkt/common"
	"github.com/coreos/rkt/tests/testutils"
)

func TestGC(t *testing.T) {
	ctx := testutils.NewRktRunCtx()
	defer ctx.Cleanup()

	// Finished pods.
	patchImportAndRun("inspect-gc-test-run.aci", []string{"--exec=/inspect --print-msg=HELLO_API --exit-code=0"}, t, ctx)

	// Prepared pods.
	patchImportAndPrepare("inspect-gc-test-prepare.aci", []string{"--exec=/inspect --print-msg=HELLO_API --exit-code=0"}, t, ctx)

	// Abort prepare.
	imagePath := patchTestACI("inspect-gc-test-abort.aci", []string{"--exec=/inspect --print-msg=HELLO_API --exit-code=0"}...)
	defer os.Remove(imagePath)
	cmd := fmt.Sprintf("%s --insecure-options=image prepare %s %s", ctx.Cmd(), imagePath, imagePath)
	spawnAndWaitOrFail(t, cmd, 1)

	gcCmd := fmt.Sprintf("%s gc --mark-only=true --expire-prepared=0 --grace-period=0", ctx.Cmd())
	spawnAndWaitOrFail(t, gcCmd, 0)

	pods := podsRemaining(t, ctx)
	if len(pods) == 0 {
		t.Fatalf("pods should still be present in rkt's data directory")
	}

	gcCmd = fmt.Sprintf("%s gc --mark-only=false --expire-prepared=0 --grace-period=0", ctx.Cmd())
	spawnAndWaitOrFail(t, gcCmd, 0)

	pods = podsRemaining(t, ctx)
	if len(pods) != 0 {
		t.Fatalf("no pods should exist rkt's data directory, but found: %v", pods)
	}
}

func podsRemaining(t *testing.T, ctx *testutils.RktRunCtx) []os.FileInfo {
	gcDirs := []string{
		filepath.Join(ctx.DataDir(), "pods", "exited-garbage"),
		filepath.Join(ctx.DataDir(), "pods", "prepared"),
		filepath.Join(ctx.DataDir(), "pods", "garbage"),
		filepath.Join(ctx.DataDir(), "pods", "run"),
	}

	var remainingPods []os.FileInfo
	for _, dir := range gcDirs {
		pods, err := ioutil.ReadDir(dir)
		if err != nil {
			t.Fatalf("cannot read gc directory %q: %v", dir, err)
		}
		remainingPods = append(remainingPods, pods...)
	}

	return remainingPods
}

func TestGCAfterUnmount(t *testing.T) {
	if !common.SupportsOverlay() {
		t.Skip("Overlay fs not supported.")
	}

	ctx := testutils.NewRktRunCtx()
	defer ctx.Cleanup()

	for _, rmNetns := range []bool{false, true} {

		imagePath := patchImportAndFetchHash("inspect-gc-test-run.aci", []string{"--exec=/inspect --print-msg=HELLO_API --exit-code=0"}, t, ctx)
		defer os.Remove(imagePath)
		cmd := fmt.Sprintf("%s --insecure-options=image prepare %s", ctx.Cmd(), imagePath)
		uuid := runRktAndGetUUID(t, cmd)

		cmd = fmt.Sprintf("%s run-prepared %s", ctx.Cmd(), uuid)
		runRktAndCheckOutput(t, cmd, "", false)

		podDir := filepath.Join(ctx.DataDir(), "pods", "run", uuid)
		stage1MntPath := filepath.Join(podDir, "stage1", "rootfs")
		stage2MntPath := filepath.Join(stage1MntPath, "opt", "stage2", "rkt-inspect", "rootfs")
		netnsPath := filepath.Join(podDir, "netns")
		podNetNSPathBytes, err := ioutil.ReadFile(netnsPath)
		if err != nil {
			t.Fatalf(`cannot read "netns" stage1: %v`, err)
		}
		podNetNSPath := string(podNetNSPathBytes)

		if err := syscall.Unmount(stage2MntPath, 0); err != nil {
			t.Fatalf("cannot umount stage2: %v", err)
		}

		if err := syscall.Unmount(stage1MntPath, 0); err != nil {
			t.Fatalf("cannot umount stage1: %v", err)
		}

		if err := syscall.Unmount(podNetNSPath, 0); err != nil {
			t.Fatalf("cannot umount pod netns: %v", err)
		}

		if rmNetns {
			_ = os.RemoveAll(podNetNSPath)
		}

		pods := podsRemaining(t, ctx)
		if len(pods) == 0 {
			t.Fatalf("pods should still be present in rkt's data directory")
		}

		gcCmd := fmt.Sprintf("%s gc --mark-only=false --expire-prepared=0 --grace-period=0", ctx.Cmd())
		// check we don't get any output (an error) after "executing net-plugin..."
		runRktAndCheckRegexOutput(t, gcCmd, `executing net-plugin .*\n\z`)

		pods = podsRemaining(t, ctx)
		if len(pods) != 0 {
			t.Fatalf("no pods should exist rkt's data directory, but found: %v", pods)
		}

	}
}
