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
	"testing"

	"github.com/rkt/rkt/common"
	"github.com/rkt/rkt/pkg/aci/acitest"
	"github.com/rkt/rkt/tests/testutils"

	"github.com/appc/spec/schema"
	"github.com/appc/spec/schema/types"
)

// TestRunConflictingFlags tests that 'rkt run' will complain and abort
// if conflicting flags are specified together with a pod manifest.
func TestRunConflictingFlags(t *testing.T) {
	ctx := testutils.NewRktRunCtx()
	defer ctx.Cleanup()

	runConflictingFlagsMsg := "conflicting flags set with --pod-manifest (see --help)"
	podManifestFlag := "--pod-manifest=/dev/null"
	conflictingFlags := []struct {
		flag string
		args string
	}{
		{"--inherit-env", ""},
		{"--no-store", ""},
		{"--store-only", ""},
		{"--port=", "foo:80"},
		{"--set-env=", "foo=bar"},
		{"--volume=", "foo,kind=host,source=/tmp"},
		{"--mount=", "volume=foo,target=/tmp --volume=foo,kind=host,source=/tmp"},
	}
	imageConflictingFlags := []struct {
		flag string
		args string
	}{
		{"--exec=", "/bin/sh"},
		{"--user=", "user_foo"},
		{"--group=", "group_foo"},
	}

	for _, cf := range conflictingFlags {
		cmd := fmt.Sprintf("%s run %s %s%s", ctx.Cmd(), podManifestFlag, cf.flag, cf.args)
		runRktAndCheckOutput(t, cmd, runConflictingFlagsMsg, true)
	}
	for _, icf := range imageConflictingFlags {
		cmd := fmt.Sprintf("%s run dummy-image.aci %s %s%s", ctx.Cmd(), podManifestFlag, icf.flag, icf.args)
		runRktAndCheckOutput(t, cmd, runConflictingFlagsMsg, true)
	}
}

// TestPreStart tests that pre-start events are run, and they run as root even
// when the app itself runs as an unprivileged user.
func TestPreStart(t *testing.T) {
	prestartManifest := schema.ImageManifest{
		Name: "coreos.com/rkt-prestart-test",
		App: &types.App{
			Exec: types.Exec{"/inspect"},
			User: "1000", Group: "1000",
			WorkingDirectory: "/",
			EventHandlers: []types.EventHandler{
				{"pre-start", types.Exec{
					"/inspect",
					"--print-user",
				}},
			},
		},
		Labels: types.Labels{
			{"version", "1.29.0"},
			{"arch", common.GetArch()},
			{"os", common.GetOS()},
		},
	}

	prestartManifestStr, err := acitest.ImageManifestString(&prestartManifest)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	prestartManifestFile := "prestart-manifest.json"
	if err := ioutil.WriteFile(prestartManifestFile, []byte(prestartManifestStr), 0600); err != nil {
		t.Fatalf("Cannot write prestart manifest: %v", err)
	}
	defer os.Remove(prestartManifestFile)
	prestartImage := patchTestACI("rkt-prestart.aci", fmt.Sprintf("--manifest=%s", prestartManifestFile))
	defer os.Remove(prestartImage)

	ctx := testutils.NewRktRunCtx()
	defer ctx.Cleanup()

	rktCmd := fmt.Sprintf("%s --insecure-options=image run %s", ctx.Cmd(), prestartImage)
	expectedLine := "User: uid=0 euid=0 gid=0 egid=0"
	runRktAndCheckOutput(t, rktCmd, expectedLine, false)
}
