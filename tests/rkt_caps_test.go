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

// +build host coreos src kvm

package main

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/coreos/rkt/tests/testutils"
	"github.com/syndtr/gocapability/capability"
)

func TestCapsSeveralApp(t *testing.T) {
	// All the following images are launched together in the same pod
	var appCapsTests = []struct {
		// constants
		testName     string // name of the image
		capRemainSet string // if != "x", value passed to actool patch-manifest --capability=
		capRemoveSet string // if != "x", value passed to actool patch-manifest --revoke-capability=
		expected     string // caps bounding set as printed by gocapability

		// set during the test
		imageFile string
	}{
		// Testing without isolators
		{
			testName:     "image-none",
			capRemainSet: "x",
			capRemoveSet: "x",
			expected: strings.Join([]string{
				"chown",
				"dac_override",
				"fowner",
				"fsetid",
				"kill",
				"setgid",
				"setuid",
				"setpcap",
				"net_bind_service",
				"net_raw",
				"sys_chroot",
				"mknod",
				"audit_write",
				"setfcap",
			}, ", "),
		},
		// Testing remain set
		{
			testName:     "image-only-one-cap",
			capRemainSet: "CAP_NET_ADMIN",
			capRemoveSet: "x",
			expected:     "net_admin",
		},
		{
			testName:     "image-only-one-cap-from-default",
			capRemainSet: "CAP_CHOWN",
			capRemoveSet: "x",
			expected:     "chown",
		},
		{
			testName: "image-some-caps",
			capRemainSet: strings.Join([]string{
				"CAP_CHOWN",
				"CAP_FOWNER",
				"CAP_SYS_ADMIN",
				"CAP_NET_ADMIN",
			}, ","),
			capRemoveSet: "x",
			expected: strings.Join([]string{
				"chown",
				"fowner",
				"net_admin",
				"sys_admin",
			}, ", "),
		},
		{
			testName: "image-caps-from-nspawn-default",
			capRemainSet: strings.Join([]string{
				"CAP_CHOWN",
				"CAP_DAC_OVERRIDE",
				"CAP_DAC_READ_SEARCH",
				"CAP_FOWNER",
				"CAP_FSETID",
				"CAP_IPC_OWNER",
				"CAP_KILL",
				"CAP_LEASE",
				"CAP_LINUX_IMMUTABLE",
				"CAP_NET_BIND_SERVICE",
				"CAP_NET_BROADCAST",
				"CAP_NET_RAW",
				"CAP_SETGID",
				"CAP_SETFCAP",
				"CAP_SETPCAP",
				"CAP_SETUID",
				"CAP_SYS_ADMIN",
				"CAP_SYS_CHROOT",
				"CAP_SYS_NICE",
				"CAP_SYS_PTRACE",
				"CAP_SYS_TTY_CONFIG",
				"CAP_SYS_RESOURCE",
				"CAP_SYS_BOOT",
				"CAP_AUDIT_WRITE",
				"CAP_AUDIT_CONTROL",
			}, ","),
			capRemoveSet: "x",
			expected: strings.Join([]string{
				"chown",
				"dac_override",
				"dac_read_search",
				"fowner",
				"fsetid",
				"kill",
				"setgid",
				"setuid",
				"setpcap",
				"linux_immutable",
				"net_bind_service",
				"net_broadcast",
				"net_raw",
				"ipc_owner",
				"sys_chroot",
				"sys_ptrace",
				"sys_admin",
				"sys_boot",
				"sys_nice",
				"sys_resource",
				"sys_tty_config",
				"lease",
				"audit_write",
				"audit_control",
				"setfcap",
			}, ", "),
		},
		// Testing revoke set
		{
			testName:     "image-revoke-one-from-default",
			capRemainSet: "x",
			capRemoveSet: "CAP_CHOWN",
			expected: strings.Join([]string{
				"dac_override, fowner, fsetid, kill, setgid, setuid, setpcap, net_bind_service, net_raw, sys_chroot, mknod, audit_write, setfcap",
			}, ", "),
		},
		{
			testName:     "image-revoke-one-already-revoked",
			capRemainSet: "x",
			capRemoveSet: "CAP_SYS_ADMIN",
			expected: strings.Join([]string{
				"chown",
				"dac_override",
				"fowner",
				"fsetid",
				"kill",
				"setgid",
				"setuid",
				"setpcap",
				"net_bind_service",
				"net_raw",
				"sys_chroot",
				"mknod",
				"audit_write",
				"setfcap",
			}, ", "),
		},
		{
			testName:     "image-revoke-two",
			capRemainSet: "x",
			capRemoveSet: "CAP_CHOWN,CAP_SYS_ADMIN",
			expected: strings.Join([]string{
				"dac_override",
				"fowner",
				"fsetid",
				"kill",
				"setgid",
				"setuid",
				"setpcap",
				"net_bind_service",
				"net_raw",
				"sys_chroot",
				"mknod",
				"audit_write",
				"setfcap",
			}, ", "),
		},
		{
			testName:     "image-revoke-all",
			capRemainSet: "x",
			capRemoveSet: strings.Join([]string{
				"CAP_AUDIT_WRITE",
				"CAP_CHOWN",
				"CAP_DAC_OVERRIDE",
				"CAP_FSETID",
				"CAP_FOWNER",
				"CAP_KILL",
				"CAP_MKNOD",
				"CAP_NET_RAW",
				"CAP_NET_BIND_SERVICE",
				"CAP_SETUID",
				"CAP_SETGID",
				"CAP_SETPCAP",
				"CAP_SETFCAP",
				"CAP_SYS_CHROOT",
			}, ","),
			expected: "",
		},
		{
			testName:     "image-revoke-all-but-one",
			capRemainSet: "x",
			capRemoveSet: strings.Join([]string{
				"CAP_AUDIT_WRITE",
				"CAP_CHOWN",
				"CAP_DAC_OVERRIDE",
				"CAP_FSETID",
				"CAP_FOWNER",
				"CAP_KILL",
				"CAP_MKNOD",
				"CAP_NET_RAW",
				"CAP_NET_BIND_SERVICE",
				"CAP_SETUID",
				"CAP_SETGID",
				"CAP_SETPCAP",
				"CAP_SETFCAP",
			}, ","),
			expected: "sys_chroot",
		},
		{
			testName:     "image-revoke-all-plus-one",
			capRemainSet: "x",
			capRemoveSet: strings.Join([]string{
				"CAP_AUDIT_WRITE",
				"CAP_CHOWN",
				"CAP_DAC_OVERRIDE",
				"CAP_FSETID",
				"CAP_FOWNER",
				"CAP_KILL",
				"CAP_MKNOD",
				"CAP_NET_RAW",
				"CAP_NET_BIND_SERVICE",
				"CAP_SETUID",
				"CAP_SETGID",
				"CAP_SETPCAP",
				"CAP_SETFCAP",
				"CAP_SYS_CHROOT",
				"CAP_SYS_ADMIN",
			}, ","),
			expected: "",
		},
		// Testing with an empty remain set or an empty remove set
		// TODO(alban): "actool patch-manifest" cannot generate those images for now
		//{
		//	testName:     "image-remain-set-empty",
		//	capRemainSet: "",
		//	capRemoveSet: "x",
		//	expected:     "",
		//},
		//{
		//	testName:     "image-revoke-none",
		//	capRemainSet: "x",
		//	capRemoveSet: "",
		//	expected:     "TODO(alban)",
		//},
	}

	ctx := testutils.NewRktRunCtx()
	defer ctx.Cleanup()

	for i, tt := range appCapsTests {
		patches := []string{
			fmt.Sprintf("--name=%s", tt.testName),
			fmt.Sprintf("--exec=/inspect --print-caps-pid=0 --suffix-msg=%s", tt.testName),
		}
		if tt.capRemainSet != "x" {
			patches = append(patches, "--capability="+tt.capRemainSet)
		}
		if tt.capRemoveSet != "x" {
			patches = append(patches, "--revoke-capability="+tt.capRemoveSet)
		}
		imageFile := patchTestACI(tt.testName+".aci", patches...)
		defer os.Remove(imageFile)
		appCapsTests[i].imageFile = imageFile
		t.Logf("Built image %q", imageFile)
	}

	// Generate the rkt arguments to launch all the apps in the same pod
	rktArgs := ""
	for _, tt := range appCapsTests {
		rktArgs += " " + tt.imageFile
	}
	cmd := fmt.Sprintf("%s --insecure-options=image run %s", ctx.Cmd(), rktArgs)

	// Ideally, the test would run the pod only one time, but all
	// apps' output is mixed together without ordering guarantees, so
	// it makes it impossible to call all the expectWithOutput() in
	// the correct order.
	for _, tt := range appCapsTests {
		t.Logf("Checking caps for %q", tt.testName)
		child := spawnOrFail(t, cmd)

		expected := fmt.Sprintf("Capability set: bounding: %s (%s)",
			tt.expected, tt.testName)
		if err := expectWithOutput(child, expected); err != nil {
			t.Fatalf("Expected %q but not found: %v", expected, err)
		}

		waitOrFail(t, child, 0)

		ctx.RunGC()
	}
}

var capsTests = []struct {
	testName            string
	capIsolator         string
	capa                capability.Cap
	capInStage1Expected bool
	capInStage2Expected bool
	nonrootCapExpected  bool
}{
	{
		testName:            "Check we don't have CAP_NET_ADMIN without isolator",
		capIsolator:         "",
		capa:                capability.CAP_NET_ADMIN,
		capInStage1Expected: false,
		capInStage2Expected: false,
		nonrootCapExpected:  false,
	},
	{
		testName:            "Check we have CAP_MKNOD without isolator",
		capIsolator:         "",
		capa:                capability.CAP_MKNOD,
		capInStage1Expected: true,
		capInStage2Expected: true,
		nonrootCapExpected:  true,
	},
	{
		testName:            "Check we have CAP_NET_ADMIN with an isolator",
		capIsolator:         "CAP_NET_ADMIN,CAP_NET_BIND_SERVICE",
		capa:                capability.CAP_NET_ADMIN,
		capInStage1Expected: true,
		capInStage2Expected: true,
		nonrootCapExpected:  true,
	},
	{
		testName:            "Check we have CAP_NET_BIND_SERVICE with an isolator",
		capIsolator:         "CAP_NET_ADMIN,CAP_NET_BIND_SERVICE",
		capa:                capability.CAP_NET_BIND_SERVICE,
		capInStage1Expected: true,
		capInStage2Expected: true,
		nonrootCapExpected:  true,
	},
	{
		testName:            "Check we don't have CAP_NET_ADMIN with an isolator setting CAP_NET_BIND_SERVICE",
		capIsolator:         "CAP_NET_BIND_SERVICE",
		capa:                capability.CAP_NET_ADMIN,
		capInStage1Expected: false,
		capInStage2Expected: false,
		nonrootCapExpected:  false,
	},
}

// CommonTestCaps creates a new capabilities test fixture for the given stages.
func NewCapsTest(hasStage1FullCaps bool, stages []int) testutils.Test {
	return testutils.TestFunc(func(t *testing.T) {
		ctx := testutils.NewRktRunCtx()
		defer ctx.Cleanup()

		for i, tt := range capsTests {
			stage1Args := []string{"--exec=/inspect --print-caps-pid=1 --print-user"}
			stage2Args := []string{"--exec=/inspect --print-caps-pid=0 --print-user"}
			if tt.capIsolator != "" {
				stage1Args = append(stage1Args, "--capability="+tt.capIsolator)
				stage2Args = append(stage2Args, "--capability="+tt.capIsolator)
			}
			stage1FileName := patchTestACI("rkt-inspect-print-caps-stage1.aci", stage1Args...)
			defer os.Remove(stage1FileName)
			stage2FileName := patchTestACI("rkt-inspect-print-caps-stage2.aci", stage2Args...)
			defer os.Remove(stage2FileName)
			stageFileNames := []string{stage1FileName, stage2FileName}

			for _, stage := range stages {
				t.Logf("Running test #%v: %v [stage %v]", i, tt.testName, stage)

				cmd := fmt.Sprintf("%s --debug --insecure-options=image run --mds-register=false --set-env=CAPABILITY=%d %s", ctx.Cmd(), int(tt.capa), stageFileNames[stage-1])
				child := spawnOrFail(t, cmd)

				expectedLine := tt.capa.String()

				capInStage1Expected := tt.capInStage1Expected || hasStage1FullCaps

				if (stage == 1 && capInStage1Expected) || (stage == 2 && tt.capInStage2Expected) {
					expectedLine += "=enabled"
				} else {
					expectedLine += "=disabled"
				}

				if err := expectWithOutput(child, expectedLine); err != nil {
					t.Fatalf("Expected %q but not found: %v", expectedLine, err)
				}

				if err := expectWithOutput(child, "User: uid=0 euid=0 gid=0 egid=0"); err != nil {
					t.Fatalf("Expected user 0 but not found: %v", err)
				}

				waitOrFail(t, child, 0)
			}
			ctx.Reset()
		}
	})
}

func TestCapsNonRoot(t *testing.T) {
	ctx := testutils.NewRktRunCtx()
	defer ctx.Cleanup()

	for i, tt := range capsTests {
		args := []string{"--exec=/inspect --print-caps-pid=0 --print-user", "--user=9000", "--group=9000"}
		if tt.capIsolator != "" {
			args = append(args, "--capability="+tt.capIsolator)
		}
		fileName := patchTestACI("rkt-inspect-print-caps-nonroot.aci", args...)
		defer os.Remove(fileName)

		t.Logf("Running test #%v: %v [non-root]", i, tt.testName)

		cmd := fmt.Sprintf("%s --debug --insecure-options=image run --mds-register=false --set-env=CAPABILITY=%d %s", ctx.Cmd(), int(tt.capa), fileName)
		child := spawnOrFail(t, cmd)

		expectedLine := tt.capa.String()
		if tt.nonrootCapExpected {
			expectedLine += "=enabled"
		} else {
			expectedLine += "=disabled"
		}
		if err := expectWithOutput(child, expectedLine); err != nil {
			t.Fatalf("Expected %q but not found: %v", expectedLine, err)
		}

		if err := expectWithOutput(child, "User: uid=9000 euid=9000 gid=9000 egid=9000"); err != nil {
			t.Fatalf("Expected user 9000 but not found: %v", err)
		}

		waitOrFail(t, child, 0)
		ctx.Reset()
	}
}
