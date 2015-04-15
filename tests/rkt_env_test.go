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
	rktCmd string
	expect string
}{
	{
		`../bin/rkt --debug --insecure-skip-verify run ./rkt-inspect-print-var-from-manifest.aci`,
		"VAR_FROM_MANIFEST=manifest",
	},
	{
		`../bin/rkt --debug --insecure-skip-verify run --set-env=VAR_OTHER=setenv ./rkt-inspect-print-var-other.aci`,
		"VAR_OTHER=setenv",
	},
	{
		`../bin/rkt --debug --insecure-skip-verify run --set-env=VAR_FROM_MANIFEST=setenv ./rkt-inspect-print-var-from-manifest.aci`,
		"VAR_FROM_MANIFEST=setenv",
	},
	{
		`/bin/sh -c "export VAR_OTHER=host ; ../bin/rkt --debug --insecure-skip-verify run --inherit-env=true ./rkt-inspect-print-var-other.aci"`,
		"VAR_OTHER=host",
	},
	{
		`/bin/sh -c "export VAR_FROM_MANIFEST=host ; ../bin/rkt --debug --insecure-skip-verify run --inherit-env=true ./rkt-inspect-print-var-from-manifest.aci"`,
		"VAR_FROM_MANIFEST=manifest",
	},
	{
		`/bin/sh -c "export VAR_OTHER=host ; ../bin/rkt --debug --insecure-skip-verify run --inherit-env=true --set-env=VAR_OTHER=setenv ./rkt-inspect-print-var-other.aci"`,
		"VAR_OTHER=setenv",
	},
}

func TestEnv(t *testing.T) {
	patchTestACI("rkt-inspect-print-var-from-manifest.aci", "--exec=/inspect --print-env=VAR_FROM_MANIFEST")
	defer os.Remove("rkt-inspect-print-var-from-manifest.aci")
	patchTestACI("rkt-inspect-print-var-other.aci", "--exec=/inspect --print-env=VAR_OTHER")
	defer os.Remove("rkt-inspect-print-var-other.aci")

	for i, tt := range envTests {
		t.Logf("Running test #%v: %v", i, tt.rktCmd)

		child, err := gexpect.Spawn(tt.rktCmd)
		if err != nil {
			t.Fatalf("Cannot exec rkt #%v: %v", i, err)
		}

		err = child.Expect(tt.expect)
		if err != nil {
			t.Fatalf("Expected %q but not found", tt.expect)
		}

		err = child.Wait()
		if err != nil {
			t.Fatalf("rkt didn't terminate correctly: %v", err)
		}
	}
}
