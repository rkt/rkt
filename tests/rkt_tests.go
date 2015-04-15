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
	"os/exec"
	"testing"
)

const unsafeEnvVar = "RKT_ENABLE_DESTRUCTIVE_TESTS"

func skipUnsafe(t *testing.T) {
	if v := os.Getenv(unsafeEnvVar); v != "1" {
		t.Skipf("%s envvar is not specified or has value different than 1 (%s), skipping the test", unsafeEnvVar, v)
	}
}

func removeDataDir(t *testing.T) {
	if err := os.RemoveAll("/var/lib/rkt"); err != nil {
		t.Fatalf("Failed to remove /var/lib/rkt: %v", err)
	}
}

func patchTestACI(newFileName string, args ...string) {
	var allArgs []string
	allArgs = append(allArgs, "patch-manifest")
	allArgs = append(allArgs, "--overwrite")
	allArgs = append(allArgs, args...)
	allArgs = append(allArgs, "rkt-inspect.aci")
	allArgs = append(allArgs, newFileName)

	output, err := exec.Command("../bin/actool", allArgs...).CombinedOutput()
	if err != nil {
		panic(fmt.Sprintf("Cannot create ACI: %v: %v\n", err, output))
	}
}
