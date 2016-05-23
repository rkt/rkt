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
	"testing"

	"github.com/coreos/rkt/tests/testutils"
)

func TestRm(t *testing.T) {
	ctx := testutils.NewRktRunCtx()
	defer ctx.Cleanup()

	var uuids []string

	img := patchTestACI("inspect-rm-test-run.aci", []string{"--exec=/inspect --print-msg=HELLO_API --exit-code=0"}...)
	defer os.Remove(img)
	prepareCmd := fmt.Sprintf("%s --insecure-options=image prepare %s", ctx.Cmd(), img)

	// Finished pod.
	uuid := runRktAndGetUUID(t, prepareCmd)
	runPreparedCmd := fmt.Sprintf("%s --insecure-options=image run-prepared %s", ctx.Cmd(), uuid)
	runRktAndCheckOutput(t, runPreparedCmd, "", false)

	uuids = append(uuids, uuid)

	// Prepared pod.
	uuid = runRktAndGetUUID(t, prepareCmd)
	uuids = append(uuids, uuid)

	podDirs := []string{
		filepath.Join(ctx.DataDir(), "pods", "run"),
		filepath.Join(ctx.DataDir(), "pods", "prepared"),
	}

	for _, dir := range podDirs {
		pods, err := ioutil.ReadDir(dir)
		if err != nil {
			t.Fatalf("cannot read pods directory %q: %v", dir, err)
		}
		if len(pods) == 0 {
			t.Fatalf("pods should still exist in directory %q: %v", dir, pods)
		}
	}

	for _, u := range uuids {
		cmd := fmt.Sprintf("%s rm %s", ctx.Cmd(), u)
		spawnAndWaitOrFail(t, cmd, 0)
	}

	podDirs = append(podDirs, filepath.Join(ctx.DataDir(), "pods", "exited-garbage"))

	for _, dir := range podDirs {
		pods, err := ioutil.ReadDir(dir)
		if err != nil {
			t.Fatalf("cannot read pods directory %q: %v", dir, err)
		}
		if len(pods) != 0 {
			t.Errorf("no pods should exist in directory %q, but found: %v", dir, pods)
		}
	}
}
