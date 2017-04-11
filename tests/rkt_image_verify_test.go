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

// +build host coreos src kvm

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/rkt/rkt/tests/testutils"
)

// TestImageVerify tests 'rkt image verify'.
// It renders an image, deletes a file from it, and verifies that 'verify' notices and repairs it.
func TestImageVerify(t *testing.T) {
	tmpDir := mustTempDir("rkt-TestImageVerify-")
	defer os.RemoveAll(tmpDir)

	ctx := testutils.NewRktRunCtx()
	defer ctx.Cleanup()

	img := getInspectImagePath()
	imgHash, err := importImageAndFetchHash(t, ctx, "--insecure-options=image", img)
	if err != nil {
		t.Fatalf("unable to fetch image: %v", err)
	}
	prepareCmd := fmt.Sprintf("%s prepare %s", ctx.Cmd(), imgHash)
	_ = runRktAndGetUUID(t, prepareCmd)

	treestorePath := filepath.Join(ctx.DataDir(), "cas", "tree")
	numDeleted := 0
	err = filepath.Walk(treestorePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.Name() == "inspect" {
			os.Remove(path)
			numDeleted++
		}
		return nil
	})
	if err != nil {
		t.Fatalf("unable to walk treestore %v: %v", treestorePath, err)
	}
	if numDeleted != 1 {
		t.Fatalf("expected to find one 'inspect' binary in the tree to delete")
	}

	cmd := fmt.Sprintf("%s image verify %s", ctx.Cmd(), imgHash)
	if err := runRktAndCheckRegexOutput(t, cmd, "tree cache verification failed for image.*"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	// treestore should be fixed so this should now be successful
	if err := runRktAndCheckRegexOutput(t, cmd, "successfully verified checksum for image.*"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
