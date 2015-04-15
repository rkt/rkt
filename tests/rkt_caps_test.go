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
	"fmt"
	"os"
	"testing"

	"github.com/coreos/rkt/Godeps/_workspace/src/github.com/ThomasRooney/gexpect"
	"github.com/coreos/rkt/Godeps/_workspace/src/github.com/syndtr/gocapability/capability"
)

var capsTests = []struct {
	testName            string
	capIsolator         string
	capa                capability.Cap
	capInStage1Expected bool
	capInStage2Expected bool
}{
	{
		testName:            "Check we don't have CAP_NET_ADMIN without isolator",
		capIsolator:         "",
		capa:                capability.CAP_NET_ADMIN,
		capInStage1Expected: false,
		capInStage2Expected: false,
	},
	{
		testName:            "Check we have CAP_MKNOD without isolator",
		capIsolator:         "",
		capa:                capability.CAP_MKNOD,
		capInStage1Expected: true,
		capInStage2Expected: true,
	},
	{
		testName:            "Check we have CAP_NET_ADMIN with an isolator",
		capIsolator:         "CAP_NET_ADMIN,CAP_NET_BIND_SERVICE",
		capa:                capability.CAP_NET_ADMIN,
		capInStage1Expected: true,
		capInStage2Expected: true,
	},
	{
		testName:            "Check we have CAP_NET_BIND_SERVICE with an isolator",
		capIsolator:         "CAP_NET_ADMIN,CAP_NET_BIND_SERVICE",
		capa:                capability.CAP_NET_BIND_SERVICE,
		capInStage1Expected: true,
		capInStage2Expected: true,
	},
	{
		testName:            "Check we don't have CAP_NET_ADMIN with an isolator setting CAP_NET_BIND_SERVICE",
		capIsolator:         "CAP_NET_BIND_SERVICE",
		capa:                capability.CAP_NET_ADMIN,
		capInStage1Expected: false,
		capInStage2Expected: false,
	},
}

func TestCaps(t *testing.T) {
	for i, tt := range capsTests {
		var stage1FileName = "rkt-inspect-print-caps-stage1.aci"
		var stage2FileName = "rkt-inspect-print-caps-stage2.aci"
		var stage1Args []string
		var stage2Args []string
		if tt.capIsolator == "" {
			stage1Args = []string{"--exec=/inspect --print-caps-pid=1"}
			stage2Args = []string{"--exec=/inspect --print-caps-pid=0"}
		} else {
			stage1Args = []string{"--capability=" + tt.capIsolator, "--exec=/inspect --print-caps-pid=1"}
			stage2Args = []string{"--capability=" + tt.capIsolator, "--exec=/inspect --print-caps-pid=0"}
		}
		patchTestACI(stage1FileName, stage1Args...)
		patchTestACI(stage2FileName, stage2Args...)
		defer os.Remove(stage1FileName)
		defer os.Remove(stage2FileName)

		for _, stage := range []int{1, 2} {
			t.Logf("Running test #%v: %v [stage %v]", i, tt.testName, stage)

			cmd := fmt.Sprintf("../bin/rkt --debug --insecure-skip-verify run --set-env=CAPABILITY=%d ./rkt-inspect-print-caps-stage%d.aci", int(tt.capa), stage)
			t.Logf("Command: %v", cmd)
			child, err := gexpect.Spawn(cmd)
			if err != nil {
				t.Fatalf("Cannot exec rkt #%v: %v", i, err)
			}

			expectedLine := tt.capa.String()
			if (stage == 1 && tt.capInStage1Expected) || (stage == 2 && tt.capInStage2Expected) {
				expectedLine += "=enabled"
			} else {
				expectedLine += "=disabled"
			}
			err = child.Expect(expectedLine)
			if err != nil {
				t.Fatalf("Expected %q but not found", expectedLine)
			}

			err = child.Wait()
			if err != nil {
				t.Fatalf("rkt didn't terminate correctly: %v", err)
			}
		}
	}
}
