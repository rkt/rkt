// Copyright 2015 The rkt Authors
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
	"testing"

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

	gcDirs := []string{
		filepath.Join(ctx.DataDir(), "pods", "exited-garbage"),
		filepath.Join(ctx.DataDir(), "pods", "prepared"),
		filepath.Join(ctx.DataDir(), "pods", "garbage"),
	}

	for _, dir := range gcDirs {
		pods, err := ioutil.ReadDir(dir)
		if err != nil {
			t.Fatalf("cannot read gc directory %q: %v", dir, err)
		}
		if len(pods) == 0 {
			t.Fatalf("pods should still exist in directory %q", dir)
		}
	}

	gcCmd = fmt.Sprintf("%s gc --mark-only=false --expire-prepared=0 --grace-period=0", ctx.Cmd())
	spawnAndWaitOrFail(t, gcCmd, 0)

	for _, dir := range gcDirs {
		pods, err := ioutil.ReadDir(dir)
		if err != nil {
			t.Fatalf("cannot read gc directory %q: %v", dir, err)
		}
		if len(pods) != 0 {
			t.Fatalf("no pods should exist in directory %q, but found: %v", dir, pods)
		}
	}
}
