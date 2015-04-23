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
	"os"
	"testing"

	"github.com/coreos/rkt/Godeps/_workspace/src/github.com/ThomasRooney/gexpect"
)

var envTests = []struct {
	runCmd    string
	runExpect string
	sleepCmd  string
	enterCmd  string
}{
	{
		`../bin/rkt --debug --insecure-skip-verify run ./rkt-inspect-print-var-from-manifest.aci`,
		"VAR_FROM_MANIFEST=manifest",
		`../bin/rkt --debug --insecure-skip-verify run ./rkt-inspect-sleep.aci`,
		`/bin/sh -c "../bin/rkt --debug enter $(../bin/rkt list --full|grep running|awk '{print $1}') /inspect --print-env=VAR_FROM_MANIFEST"`,
	},
	{
		`../bin/rkt --debug --insecure-skip-verify run --set-env=VAR_OTHER=setenv ./rkt-inspect-print-var-other.aci`,
		"VAR_OTHER=setenv",
		`../bin/rkt --debug --insecure-skip-verify run --set-env=VAR_OTHER=setenv ./rkt-inspect-sleep.aci`,
		`/bin/sh -c "../bin/rkt --debug enter $(../bin/rkt list --full|grep running|awk '{print $1}') /inspect --print-env=VAR_OTHER"`,
	},
	{
		`../bin/rkt --debug --insecure-skip-verify run --set-env=VAR_FROM_MANIFEST=setenv ./rkt-inspect-print-var-from-manifest.aci`,
		"VAR_FROM_MANIFEST=setenv",
		`../bin/rkt --debug --insecure-skip-verify run --set-env=VAR_FROM_MANIFEST=setenv ./rkt-inspect-sleep.aci`,
		`/bin/sh -c "../bin/rkt --debug enter $(../bin/rkt list --full|grep running|awk '{print $1}') /inspect --print-env=VAR_FROM_MANIFEST"`,
	},
	{
		`/bin/sh -c "export VAR_OTHER=host ; ../bin/rkt --debug --insecure-skip-verify run --inherit-env=true ./rkt-inspect-print-var-other.aci"`,
		"VAR_OTHER=host",
		`/bin/sh -c "export VAR_OTHER=host ; ../bin/rkt --debug --insecure-skip-verify run --inherit-env=true ./rkt-inspect-sleep.aci"`,
		`/bin/sh -c "export VAR_OTHER=host ; ../bin/rkt --debug enter $(../bin/rkt list --full|grep running|awk '{print $1}') /inspect --print-env=VAR_OTHER"`,
	},
	{
		`/bin/sh -c "export VAR_FROM_MANIFEST=host ; ../bin/rkt --debug --insecure-skip-verify run --inherit-env=true ./rkt-inspect-print-var-from-manifest.aci"`,
		"VAR_FROM_MANIFEST=manifest",
		`/bin/sh -c "export VAR_FROM_MANIFEST=host ; ../bin/rkt --debug --insecure-skip-verify run --inherit-env=true ./rkt-inspect-sleep.aci"`,
		`/bin/sh -c "export VAR_FROM_MANIFEST=host ; ../bin/rkt --debug enter $(../bin/rkt list --full|grep running|awk '{print $1}') /inspect --print-env=VAR_FROM_MANIFEST"`,
	},
	{
		`/bin/sh -c "export VAR_OTHER=host ; ../bin/rkt --debug --insecure-skip-verify run --inherit-env=true --set-env=VAR_OTHER=setenv ./rkt-inspect-print-var-other.aci"`,
		"VAR_OTHER=setenv",
		`/bin/sh -c "export VAR_OTHER=host ; ../bin/rkt --debug --insecure-skip-verify run --inherit-env=true --set-env=VAR_OTHER=setenv ./rkt-inspect-sleep.aci"`,
		`/bin/sh -c "export VAR_OTHER=host ; ../bin/rkt --debug enter $(../bin/rkt list --full|grep running|awk '{print $1}') /inspect --print-env=VAR_OTHER"`,
	},
}

func TestEnv(t *testing.T) {
	patchTestACI("rkt-inspect-print-var-from-manifest.aci", "--exec=/inspect --print-env=VAR_FROM_MANIFEST")
	defer os.Remove("rkt-inspect-print-var-from-manifest.aci")
	patchTestACI("rkt-inspect-print-var-other.aci", "--exec=/inspect --print-env=VAR_OTHER")
	defer os.Remove("rkt-inspect-print-var-other.aci")
	patchTestACI("rkt-inspect-sleep.aci", "--exec=/inspect --print-msg=Hello --sleep=84000")
	defer os.Remove("rkt-inspect-sleep.aci")

	for i, tt := range envTests {
		// 'run' tests
		t.Logf("Running 'run' test #%v: %v", i, tt.runCmd)
		child, err := gexpect.Spawn(tt.runCmd)
		if err != nil {
			t.Fatalf("Cannot exec rkt #%v: %v", i, err)
		}

		err = child.Expect(tt.runExpect)
		if err != nil {
			t.Fatalf("Expected %q but not found", tt.runExpect)
		}

		err = child.Wait()
		if err != nil {
			t.Fatalf("rkt didn't terminate correctly: %v", err)
		}

		// 'enter' tests
		t.Logf("Running 'enter' test #%v: sleep: %v", i, tt.sleepCmd)
		child, err = gexpect.Spawn(tt.sleepCmd)
		if err != nil {
			t.Fatalf("Cannot exec rkt #%v: %v", i, err)
		}

		err = child.Expect("Hello")
		if err != nil {
			t.Fatalf("Expected %q but not found", tt.runExpect)
		}

		t.Logf("Running 'enter' test #%v: enter: %v", i, tt.enterCmd)
		enterChild, err := gexpect.Spawn(tt.enterCmd)
		if err != nil {
			t.Fatalf("Cannot exec rkt #%v: %v", i, err)
		}

		err = enterChild.Expect(tt.runExpect)
		if err != nil {
			t.Fatalf("Expected %q but not found", tt.runExpect)
		}

		err = enterChild.Wait()
		if err != nil {
			t.Fatalf("rkt didn't terminate correctly: %v", err)
		}
		err = child.Close()
		if err != nil {
			t.Fatalf("rkt didn't terminate correctly: %v", err)
		}
	}
}
