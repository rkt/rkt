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

// +build host coreos src

package main

import (
	"fmt"
	"os"
	"testing"

	"github.com/coreos/rkt/tests/testutils"
)

// TestPathsWrite checks whether access to paths like /proc/sysrq-trigger are
// restricted
func TestPathsWrite(t *testing.T) {
	imageFile := patchTestACI("rkt-inspect-paths.aci",
		"--exec=/inspect --write-file --print-msg=testing-insecure-option")
	defer os.Remove(imageFile)

	ctx := testutils.NewRktRunCtx()
	defer ctx.Cleanup()

	tests := []struct {
		Path    string
		Content string
	}{
		{
			Path:    "/proc/sysrq-trigger",
			Content: "h", // Print help message in dmesg: not dangerous
		},
	}

	for _, tt := range tests {
		// Without --insecure-options=paths
		for _, insecureOption := range []string{"image", "image,ondisk,capabilities", "all-fetch"} {
			cmd := fmt.Sprintf(`%s --debug --insecure-options=%s run --set-env=FILE=%s --set-env=CONTENT=%s %s`,
				ctx.Cmd(), insecureOption, tt.Path, tt.Content, imageFile)
			t.Logf("Attempting to write on %q with --insecure-options=%s (expecting error)\n", tt.Path, insecureOption)
			expect := fmt.Sprintf("open %s: read-only file system", tt.Path)
			runRktAndCheckOutput(t, cmd, expect, true)
		}

		// With --insecure-options=paths
		for _, insecureOption := range []string{"image,paths", "image,paths,ondisk,capabilities", "image,all-run", "all"} {
			cmd := fmt.Sprintf(`%s --debug --insecure-options=%s run --set-env=FILE=%s --set-env=CONTENT=%s %s`,
				ctx.Cmd(), insecureOption, tt.Path, tt.Content, imageFile)
			t.Logf("Attempting to write on %q with --insecure-options=%s (expecting success)\n", tt.Path, insecureOption)
			runRktAndCheckOutput(t, cmd, "testing-insecure-option", false)
		}
	}
}

// TestPathsStat checks that access to inaccessible paths under
// /proc or /sys is correctly restricted:
// https://github.com/coreos/rkt/issues/2484
func TestPathsStat(t *testing.T) {
	hiddenEntry := "/sys/firmware/"
	hiddenImage := patchTestACI("rkt-inspect-stat-procfs.aci", fmt.Sprintf("--exec=/inspect --stat-file --file-name %s", hiddenEntry))
	defer os.Remove(hiddenImage)

	hiddenCtx := testutils.NewRktRunCtx()
	defer hiddenCtx.Cleanup()

	hiddenCmd := fmt.Sprintf("%s --insecure-options=image run %s", hiddenCtx.Cmd(), hiddenImage)

	hiddenExpectedLine := fmt.Sprintf("%s: mode: d---------", hiddenEntry)
	runRktAndCheckOutput(t, hiddenCmd, hiddenExpectedLine, false)
}
