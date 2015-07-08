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
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/coreos/rkt/Godeps/_workspace/src/github.com/ThomasRooney/gexpect"
)

// TestImageExtract tests 'rkt image extract', it will import some existing
// image with the inspect binary, extract it with rkt image extract and check
// that the exported /inspect hash matches the original inspect binary hash
func TestImageExtract(t *testing.T) {
	testImage := os.Getenv("RKT_INSPECT_IMAGE")
	if testImage == "" {
		panic("Empty RKT_INSPECT_IMAGE environment variable")
	}
	testImageName := "coreos.com/rkt-inspect"
	inspectFile := os.Getenv("INSPECT_BINARY")
	if inspectFile == "" {
		panic("Empty INSPECT_BINARY environment variable")
	}
	inspectHash, err := getHash(inspectFile)
	if err != nil {
		panic("Cannot get inspect binary's hash")
	}

	tmpDir, err := ioutil.TempDir("", "rkt-TestImageRender-")
	if err != nil {
		panic(fmt.Sprintf("Cannot create temp dir: %v", err))
	}
	defer os.RemoveAll(tmpDir)

	ctx := newRktRunCtx()
	defer ctx.cleanup()

	testImageShortHash := importImageAndFetchHash(t, ctx, testImage)

	tests := []struct {
		image        string
		shouldFind   bool
		expectedHash string
	}{
		{
			testImageName,
			true,
			inspectHash,
		},
		{
			testImageShortHash,
			true,
			inspectHash,
		},
		{
			"sha512-not-existed",
			false,
			"",
		},
		{
			"some~random~aci~name",
			false,
			"",
		},
	}

	for i, tt := range tests {
		outputPath := filepath.Join(tmpDir, fmt.Sprintf("extracted-%d", i))
		runCmd := fmt.Sprintf("%s image extract --rootfs-only %s %s", ctx.cmd(), tt.image, outputPath)
		t.Logf("Running 'image extract' test #%v: %v", i, runCmd)
		child, err := gexpect.Spawn(runCmd)
		if err != nil {
			t.Fatalf("Cannot exec rkt #%v: %v", i, err)
		}

		if err := child.Wait(); err != nil {
			if !tt.shouldFind && err.Error() == "exit status 1" {
				continue
			} else if tt.shouldFind || err.Error() != "exit status 1" {
				t.Fatalf("rkt didn't terminate correctly: %v", err)
			}
		}

		extractedInspectHash, err := getHash(filepath.Join(outputPath, "inspect"))
		if err != nil {
			t.Fatalf("Cannot get rendered inspect binary's hash")
		}
		if extractedInspectHash != tt.expectedHash {
			t.Fatalf("Expected /inspect hash %q but got %s", tt.expectedHash, extractedInspectHash)
		}
	}
}
