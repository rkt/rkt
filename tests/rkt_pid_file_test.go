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
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/coreos/rkt/Godeps/_workspace/src/github.com/ThomasRooney/gexpect"
)

func TestPidFileRace(t *testing.T) {
	patchTestACI("rkt-inspect-sleep.aci", "--exec=/inspect --read-stdin")
	defer os.Remove("rkt-inspect-sleep.aci")

	ctx := newRktRunCtx()
	defer ctx.cleanup()

	// Start the pod
	runCmd := fmt.Sprintf("%s --debug --insecure-skip-verify run -interactive ./rkt-inspect-sleep.aci", ctx.cmd())
	t.Logf("%s", runCmd)
	child, err := gexpect.Spawn(runCmd)
	if err != nil {
		t.Fatalf("Cannot exec rkt")
	}

	err = child.Expect("Enter text:")
	if err != nil {
		t.Fatalf("Waited for the prompt but not found: %v", err)
	}

	// Check the pid file is really created
	cmd := fmt.Sprintf(`%s list --full|grep running`, ctx.cmd())
	output, err := exec.Command("/bin/sh", "-c", cmd).CombinedOutput()
	if err != nil {
		t.Fatalf("Couldn't list the pods: %v", err)
	}
	UUID := strings.Split(string(output), "\t")[0]

	pidFileName := filepath.Join(ctx.dataDir(), "pods/run", UUID, "pid")
	if _, err := os.Stat(pidFileName); err != nil {
		t.Fatalf("Pid file missing: %v", err)
	}

	// Temporarily move the pid file away
	pidFileNameBackup := pidFileName + ".backup"
	if err := os.Rename(pidFileName, pidFileNameBackup); err != nil {
		t.Fatalf("Cannot move pid file away: %v", err)
	}

	// Start the "enter" command without the pidfile
	enterCmd := fmt.Sprintf("%s --debug enter %s /inspect --print-msg=RktEnterWorksFine", ctx.cmd(), UUID)
	t.Logf("%s", enterCmd)
	enterChild, err := gexpect.Spawn(enterCmd)
	if err != nil {
		t.Fatalf("Cannot exec rkt enter")
	}
	// Enter should be able to wait until the pid file appears
	time.Sleep(1 * time.Second)

	// Restore pid file so the "enter" command can find it
	if err := os.Rename(pidFileNameBackup, pidFileName); err != nil {
		t.Fatalf("Cannot restore pid file: %v", err)
	}

	// Now the "enter" command works and can complete
	err = enterChild.Expect("RktEnterWorksFine")
	if err != nil {
		t.Fatalf("Waited for enter to works but failed: %v", err)
	}
	err = enterChild.Wait()
	if err != nil {
		t.Fatalf("rkt enter didn't terminate correctly: %v", err)
	}

	// Terminate the pod
	err = child.SendLine("Bye")
	if err != nil {
		t.Fatalf("rkt couldn't write to the container: %v", err)
	}
	err = child.Expect("Received text: Bye")
	if err != nil {
		t.Fatalf("Expected Bye but not found: %v", err)
	}
	err = child.Wait()
	if err != nil {
		t.Fatalf("rkt didn't terminate correctly: %v", err)
	}
}
