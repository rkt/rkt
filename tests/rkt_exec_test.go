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
	"os"
	"strings"
	"testing"

	"github.com/coreos/rkt/Godeps/_workspace/src/github.com/appc/spec/schema/types"
	"github.com/coreos/rkt/Godeps/_workspace/src/github.com/steveeJ/gexpect"
)

func TestRunOverrideExec(t *testing.T) {
	execImage := patchTestACI("rkt-exec-override.aci", "--exec=/inspect")
	defer os.Remove(execImage)
	ctx := newRktRunCtx()
	defer ctx.cleanup()

	for _, tt := range []struct {
		rktCmd       string
		expectedLine string
	}{
		{
			// Sanity check - make sure no --exec override prints the expected exec invocation
			rktCmd:       fmt.Sprintf("%s --insecure-skip-verify run --mds-register=false %s -- --print-exec", ctx.cmd(), execImage),
			expectedLine: "inspect execed as: /inspect",
		},
		{
			// Now test overriding the entrypoint (which is a symlink to /inspect so should behave identically)
			rktCmd:       fmt.Sprintf("%s --insecure-skip-verify run --mds-register=false %s --exec /inspect-link -- --print-exec", ctx.cmd(), execImage),
			expectedLine: "inspect execed as: /inspect-link",
		},
	} {
		runRktAndCheckOutput(t, tt.rktCmd, tt.expectedLine)
	}
}

func TestRunPreparedOverrideExec(t *testing.T) {
	execImage := patchTestACI("rkt-exec-override.aci", "--exec=/inspect")
	defer os.Remove(execImage)
	ctx := newRktRunCtx()
	defer ctx.cleanup()

	var rktCmd, uuid, expected string

	// Sanity check - make sure no --exec override prints the expected exec invocation
	rktCmd = fmt.Sprintf("%s prepare --insecure-skip-verify %s -- --print-exec", ctx.cmd(), execImage)
	uuid = runRktAndGetUUID(t, rktCmd)

	rktCmd = fmt.Sprintf("%s run-prepared --mds-register=false %s", ctx.cmd(), uuid)
	expected = "inspect execed as: /inspect"
	runRktAndCheckOutput(t, rktCmd, expected)

	// Now test overriding the entrypoint (which is a symlink to /inspect so should behave identically)
	rktCmd = fmt.Sprintf("%s prepare --insecure-skip-verify %s --exec /inspect-link -- --print-exec", ctx.cmd(), execImage)
	uuid = runRktAndGetUUID(t, rktCmd)

	rktCmd = fmt.Sprintf("%s run-prepared --mds-register=false %s", ctx.cmd(), uuid)
	expected = "inspect execed as: /inspect-link"
	runRktAndCheckOutput(t, rktCmd, expected)
}

func runRktAndCheckOutput(t *testing.T, rktCmd, expectedLine string) {
	child, err := gexpect.Spawn(rktCmd)
	if err != nil {
		t.Fatalf("cannot exec rkt: %v", err)
	}

	if err = expectWithOutput(child, expectedLine); err != nil {
		t.Fatalf("didn't receive expected output %q: %v", expectedLine, err)
	}

	if err = child.Wait(); err != nil {
		t.Fatalf("rkt didn't terminate correctly: %v", err)
	}
}

func runRktAndGetUUID(t *testing.T, rktCmd string) string {
	child, err := gexpect.Spawn(rktCmd)
	if err != nil {
		t.Fatalf("cannot exec rkt: %v", err)
	}

	result, out, err := expectRegexWithOutput(child, "\n[0-9a-f-]{36}")
	if err != nil || len(result) != 1 {
		t.Fatalf("Error: %v\nOutput: %v", err, out)
	}

	if err = child.Wait(); err != nil {
		t.Fatalf("rkt didn't terminate correctly: %v", err)
	}

	podIDStr := strings.TrimSpace(result[0])
	podID, err := types.NewUUID(podIDStr)
	if err != nil {
		t.Fatalf("%q is not a valid UUID: %v", podIDStr, err)
	}

	return podID.String()
}
