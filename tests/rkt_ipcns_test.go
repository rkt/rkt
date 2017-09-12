// Copyright 2016-2017 The rkt Authors
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

// +build !fly,!kvm

package main

import (
	"fmt"
	"os"
	"testing"

	"github.com/rkt/rkt/tests/testutils"
)

// TestIpc test that the --ipc option works.
func TestIpc(t *testing.T) {
	imageFile := patchTestACI("rkt-inspect-ipc.aci", "--exec=/inspect --print-ipcns")
	defer os.Remove(imageFile)

	ctx := testutils.NewRktRunCtx()
	defer ctx.Cleanup()

	hostIpcNs, err := os.Readlink("/proc/self/ns/ipc")
	if err != nil {
		t.Fatalf("Cannot evaluate IPC NS symlink: %v", err)
	}

	tests := []struct {
		option string
		host   bool
		ret    int
	}{
		{
			"",
			false,
			0,
		},
		{
			"--ipc=auto",
			false,
			0,
		},
		{
			"--ipc=private",
			false,
			0,
		},
		{
			"--ipc=parent",
			true,
			0,
		},
		{
			"--ipc=supercalifragilisticexpialidocious",
			false,
			254,
		},
	}

	for _, tt := range tests {
		cmd := fmt.Sprintf("%s run --insecure-options=image %s %s", ctx.Cmd(), tt.option, imageFile)

		child := spawnOrFail(t, cmd)
		ctx.RegisterChild(child)

		expectedRegex := `IPCNS: (ipc:\[\d+\])`

		result, out, err := expectRegexWithOutput(child, expectedRegex)
		if tt.ret == 0 {
			if err != nil {
				t.Fatalf("Error: %v\nOutput: %v", err, out)
			}

			nsChanged := hostIpcNs != result[1]
			if tt.host == nsChanged {
				t.Fatalf("unexpected ipc ns %q for with option %q (host ipc ns %q)", result, tt.option, hostIpcNs)
			}
		} else {
			if err == nil {
				t.Fatalf("test %q didn't fail as expected\nOutput: %v", tt.option, out)
			}

		}

		waitOrFail(t, child, tt.ret)
	}
}
