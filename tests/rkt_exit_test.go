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
	"testing"
	"time"

	"github.com/coreos/rkt/Godeps/_workspace/src/github.com/steveeJ/gexpect"
)

func checkStatus(t *testing.T, ctx *rktRunCtx, appName, expected string) {
	cmd := fmt.Sprintf(`/bin/sh -c "`+
		`UUID=$(%s list --full|grep '^[a-f0-9]'|awk '{print $1}') ;`+
		`echo -n 'status=' ;`+
		`%s status $UUID|grep '^app-%s.*=[0-9]*$'|cut -d= -f2"`,
		ctx.cmd(), ctx.cmd(), appName)

	t.Logf("Get status for app %s: %s\n", appName, cmd)
	child, err := gexpect.Spawn(cmd)
	if err != nil {
		t.Fatalf("Cannot exec rkt")
	}

	err = expectWithOutput(child, expected)
	if err != nil {
		// For debugging purpose, print the full output of
		// "rkt list" and "rkt status"
		cmd := fmt.Sprintf(`%s list --full ;`+
			`UUID=$(%s list --full|grep  '^[a-f0-9]'|awk '{print $1}') ;`+
			`%s status $UUID`,
			ctx.cmd(), ctx.cmd(), ctx.cmd())
		out, err2 := exec.Command("/bin/sh", "-c", cmd).CombinedOutput()
		if err2 != nil {
			t.Logf("Could not run rkt status: %v. %s", err2, out)
		} else {
			t.Logf("%s\n", out)
		}

		t.Fatalf("Failed to get the status for app %s: expected: %s. %v",
			appName, expected, err)
	}

	err = child.Wait()
	if err != nil {
		t.Fatalf("rkt didn't terminate correctly: %v", err)
	}
}

// TestExitCodeSimple is testing a few exit codes on 1 pod containing just 1 app
func TestExitCodeSimple(t *testing.T) {
	for i := 0; i < 3; i++ {
		t.Logf("%d\n", i)
		imageFile := patchTestACI("rkt-inspect-exit.aci", fmt.Sprintf("--exec=/inspect --print-msg=Hello --exit-code=%d", i))
		defer os.Remove(imageFile)
		ctx := newRktRunCtx()
		defer ctx.cleanup()

		cmd := fmt.Sprintf(`%s --debug --insecure-skip-verify run --mds-register=false %s`,
			ctx.cmd(), imageFile)
		t.Logf("%s\n", cmd)
		child, err := gexpect.Spawn(cmd)
		if err != nil {
			t.Fatalf("Cannot exec rkt")
		}
		err = child.Wait()
		if err != nil {
			t.Fatalf("rkt didn't terminate correctly: %v", err)
		}

		checkStatus(t, ctx, "rkt-inspect", fmt.Sprintf("status=%d", i))
	}
}

// TestExitCodeWithSeveralApps is testing a pod with three apps returning different
// exit codes.
func TestExitCodeWithSeveralApps(t *testing.T) {
	image0File := patchTestACI("rkt-inspect-exit-0.aci", "--name=hello0",
		"--exec=/inspect --print-msg=HelloWorld --exit-code=0")
	defer os.Remove(image0File)

	image1File := patchTestACI("rkt-inspect-exit-1.aci", "--name=hello1",
		"--exec=/inspect --print-msg=HelloWorld --exit-code=1")
	defer os.Remove(image1File)

	image2File := patchTestACI("rkt-inspect-exit-2.aci", "--name=hello2",
		"--exec=/inspect --print-msg=HelloWorld --exit-code=2 --sleep=1")
	defer os.Remove(image2File)

	ctx := newRktRunCtx()
	defer ctx.cleanup()

	cmd := fmt.Sprintf(`%s --debug --insecure-skip-verify run --mds-register=false %s %s %s`,
		ctx.cmd(), image0File, image1File, image2File)
	t.Logf("%s\n", cmd)
	child, err := gexpect.Spawn(cmd)
	if err != nil {
		t.Fatalf("Cannot exec rkt")
	}

	for i := 0; i < 3; i++ {
		// The 3 apps print the same message. We don't have any ordering
		// guarantee but we don't need it.
		err = expectTimeoutWithOutput(child, "HelloWorld", time.Minute)
		if err != nil {
			t.Fatalf("Could not start the app (#%d): %v", i, err)
		}
	}

	t.Logf("Check intermediary status\n")

	// TODO: how to make sure hello0 and hello1 terminated? They should
	// terminate soon because they already printed their HelloWorld message.
	time.Sleep(100 * time.Millisecond)

	checkStatus(t, ctx, "hello0", "status=0")
	checkStatus(t, ctx, "hello1", "status=1")
	// Currently, hello2 should be stop correctly (exit code 0) when hello1
	// failed, so it cannot return its exit code 2. This might change with
	// https://github.com/coreos/rkt/issues/1461
	checkStatus(t, ctx, "hello2", "status=0")

	t.Logf("Waiting pod termination\n")
	err = child.Wait()
	if err != nil {
		t.Fatalf("rkt didn't terminate correctly: %v", err)
	}

	t.Logf("Check final status\n")

	checkStatus(t, ctx, "hello0", "status=0")
	checkStatus(t, ctx, "hello1", "status=1")
	// Currently, hello2 should be stop correctly (exit code 0) when hello1
	// failed, so it cannot return its exit code 2. This might change with
	// https://github.com/coreos/rkt/issues/1461
	checkStatus(t, ctx, "hello2", "status=0")
}
