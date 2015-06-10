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
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/coreos/rkt/Godeps/_workspace/src/github.com/ThomasRooney/gexpect"
)

const (
	// if you change this you need to change tests/image/manifest accordingly
	maxMemoryUsage = 25 * 1024 * 1024 // 25MB
)

var memoryTest = struct {
	testName     string
	aciBuildArgs []string
	rktArgs      string
}{
	`Check memory isolator 25MB`,
	[]string{"--exec=/inspect --print-memorylimit"},
	`--insecure-skip-verify run rkt-inspect-isolators.aci`,
}

func isControllerEnabled(controller string) (bool, error) {
	cg, err := os.Open("/proc/1/cgroup")
	if err != nil {
		return false, fmt.Errorf("error opening /proc/1/cgroup: %v", err)
	}
	defer cg.Close()

	s := bufio.NewScanner(cg)
	for s.Scan() {
		parts := strings.SplitN(s.Text(), ":", 2)
		if len(parts) < 2 {
			return false, fmt.Errorf("error parsing /proc/1/cgroup")
		}
		controllerParts := strings.Split(parts[1], ",")
		for _, c := range controllerParts {
			if c == controller {
				return true, nil
			}
		}
	}

	return false, nil
}

func TestAppIsolatorMemory(t *testing.T) {
	ok, err := isControllerEnabled("memory")
	if err != nil {
		t.Fatalf("Error checking if the memory cgroup controller is enabled: %v", err)
	}
	if !ok {
		t.Skip("Memory cgroup controller disabled.")
	}

	ctx := newRktRunCtx()
	defer ctx.cleanup()

	t.Logf("Running test: %v", memoryTest.testName)

	aciFileName := "rkt-inspect-isolators.aci"
	patchTestACI(aciFileName, memoryTest.aciBuildArgs...)
	defer os.Remove(aciFileName)

	rktCmd := fmt.Sprintf("%s %s", ctx.cmd(), memoryTest.rktArgs)
	t.Logf("Command: %v", rktCmd)
	child, err := gexpect.Spawn(rktCmd)
	if err != nil {
		t.Fatalf("Cannot exec rkt: %v", err)
	}
	expectedLine := "Memory Limit: " + strconv.Itoa(maxMemoryUsage)
	if err := expectWithOutput(child, expectedLine); err != nil {
		t.Fatalf("Didn't receive expected output %q: %v", expectedLine, err)
	}

	err = child.Wait()
	if err != nil {
		t.Fatalf("rkt didn't terminate correctly: %v", err)
	}
}
