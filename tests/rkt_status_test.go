// Copyright 2017 The rkt Authors
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
	"io/ioutil"
	"os"
	"testing"

	"github.com/rkt/rkt/tests/testutils"
)

func TestUuidFile(t *testing.T) {
	const imgName = "rkt-uuid-file-status-test"
	imagePath := patchTestACI(fmt.Sprintf("%s.aci", imgName), fmt.Sprintf("--name=%s", imgName))
	defer os.Remove(imagePath)

	ctx := testutils.NewRktRunCtx()
	defer ctx.Cleanup()

	cmd := fmt.Sprintf("%s --insecure-options=image prepare %s", ctx.Cmd(), imagePath)
	podUuid := runRktAndGetUUID(t, cmd)
	tmpDir := mustTempDir(imgName)
	defer os.RemoveAll(tmpDir)
	uuidFile, err := ioutil.TempFile(tmpDir, "uuid-file")
	if err != nil {
		panic(fmt.Sprintf("Cannot create uuid-file: %v", err))
	}
	uuidFilePath := uuidFile.Name()
	if err := ioutil.WriteFile(uuidFilePath, []byte(podUuid), 0600); err != nil {
		panic(fmt.Sprintf("Cannot write pod uuid to uuid-file %v", err))
	}
	statusCmd := fmt.Sprintf("%s status --uuid-file=%s", ctx.Cmd(), uuidFilePath)
	runRktAndCheckRegexOutput(t, statusCmd, "state=prepared")
}
