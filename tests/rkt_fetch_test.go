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
	"os"
	"testing"

	"github.com/coreos/rkt/Godeps/_workspace/src/github.com/steveeJ/gexpect"
)

// TestFetch tests that 'rkt fetch' will always bypass the on-disk store and
// fetch from a local file or from a remote URL.
func TestFetch(t *testing.T) {
	image := "rkt-inspect-fetch.aci"

	ctx := newRktRunCtx()
	defer ctx.cleanup()

	// Fetch the image for the first time, this should write the image to the
	// on-disk store.
	oldHash := patchImportAndFetchHash(image, []string{"--exec=/inspect --read-file"}, t, ctx)

	// Fetch the image with the same name but different content, the expecting
	// result is that we should get a different hash since we are not fetching
	// from the on-disk store.
	newHash := patchImportAndFetchHash(image, []string{"--exec=/inspect --read-file --write-file"}, t, ctx)

	if oldHash == newHash {
		t.Fatalf("ACI hash should be different as the image has changed")
	}
}

// TestRunPrepareFromFile tests that when 'rkt run/prepare' a local ACI, it will bypass the
// on-disk store.
func TestRunPrepareFromFile(t *testing.T) {
	foundMsg := "found image in local store"
	image := "rkt-inspect-implicit-fetch.aci"

	imagePath := patchTestACI(image, "--exec=/inspect")
	defer os.Remove(imagePath)

	tests := []string{
		imagePath,
		"file://" + imagePath,
	}

	ctx := newRktRunCtx()
	defer ctx.cleanup()

	importImageAndFetchHash(t, ctx, imagePath)

	for _, tt := range tests {

		// 1. Try run/prepare with '--local', should not get the $foundMsg, since we will ignore the '--local' when
		// the image is a filepath.
		cmds := []string{
			fmt.Sprintf("%s --insecure-skip-verify run --mds-register=false --local %s", ctx.cmd(), tt),
			fmt.Sprintf("%s --insecure-skip-verify prepare --local %s", ctx.cmd(), tt),
		}

		for _, cmd := range cmds {
			t.Logf("Running test %v", cmd)

			child, err := gexpect.Spawn(cmd)
			if err != nil {
				t.Fatalf("Cannot exec rkt: %v", err)
			}

			if err := child.Expect(foundMsg); err == nil {
				t.Fatalf("%q should not be found", foundMsg)
			}

			if err := child.Wait(); err != nil {
				t.Fatalf("rkt didn't terminate correctly: %v", err)
			}
		}

		// 2. Try run/prepare without '--local', should not get $foundMsg either.
		cmds = []string{
			fmt.Sprintf("%s --insecure-skip-verify run --mds-register=false %s", ctx.cmd(), tt),
			fmt.Sprintf("%s --insecure-skip-verify prepare %s", ctx.cmd(), tt),
		}

		for _, cmd := range cmds {
			t.Logf("Running test %v", cmd)

			child, err := gexpect.Spawn(cmd)
			if err != nil {
				t.Fatalf("Cannot exec rkt: %v", err)
			}
			if err := child.Expect(foundMsg); err == nil {
				t.Fatalf("%q should not be found", foundMsg)
			}
			if err := child.Wait(); err != nil {
				t.Fatalf("rkt didn't terminate correctly: %v", err)
			}
		}
	}
}

// TestImplicitFetch tests that 'rkt run/prepare' will always bypass the on-disk store
// if the tag is "latest".
func TestImplicitFetch(t *testing.T) {
	foundMsg := "found image in local store"

	ctx := newRktRunCtx()
	defer ctx.cleanup()

	// 1. Fetch the image.
	// TODO(yifan): Add other ACI with different schemes.
	importImageAndFetchHash(t, ctx, "docker://busybox:ubuntu-12.04")
	importImageAndFetchHash(t, ctx, "docker://busybox:latest")

	// 2. Try run/prepare with/without tag ':latest', should not get $foundMsg.
	cmds := []string{
		fmt.Sprintf("%s --insecure-skip-verify run --mds-register=false docker://busybox", ctx.cmd()),
		fmt.Sprintf("%s --insecure-skip-verify run --mds-register=false docker://busybox:latest", ctx.cmd()),
		fmt.Sprintf("%s --insecure-skip-verify prepare docker://busybox", ctx.cmd()),
		fmt.Sprintf("%s --insecure-skip-verify prepare docker://busybox:latest", ctx.cmd()),
	}

	for _, cmd := range cmds {
		t.Logf("Running test %v", cmd)

		child, err := gexpect.Spawn(cmd)
		if err != nil {
			t.Fatalf("Cannot exec rkt: %v", err)
		}
		if err := expectWithOutput(child, foundMsg); err == nil {
			t.Fatalf("%q should not be found", foundMsg)
		}
		if err := child.Wait(); err != nil {
			t.Fatalf("rkt didn't terminate correctly: %v", err)
		}
	}
}

// TestRunPrepareLocal tests that 'rkt run/prepare' will only use the on-disk store if flag is --local
func TestRunPrepareLocal(t *testing.T) {
	notAvailableMsg := "not available in local store"
	foundMsg := "using image in local store"

	ctx := newRktRunCtx()
	defer ctx.cleanup()

	cmds := []string{
		fmt.Sprintf("%s --insecure-skip-verify run --local --mds-register=false docker://busybox", ctx.cmd()),
		fmt.Sprintf("%s --insecure-skip-verify run --local --mds-register=false docker://busybox:latest", ctx.cmd()),
		fmt.Sprintf("%s --insecure-skip-verify prepare --local docker://busybox", ctx.cmd()),
		fmt.Sprintf("%s --insecure-skip-verify prepare --local docker://busybox:latest", ctx.cmd()),
	}

	// 1. Try run/prepare with the image not available in the store, should get $notAvailableMsg.
	for _, cmd := range cmds {
		t.Logf("Running test %v", cmd)

		child, err := gexpect.Spawn(cmd)
		if err != nil {
			t.Fatalf("Cannot exec rkt: %v", err)
		}
		if err := expectWithOutput(child, notAvailableMsg); err != nil {
			t.Fatalf("%q should be found", notAvailableMsg)
		}
		child.Wait()
	}

	// 2. Fetch the image
	importImageAndFetchHash(t, ctx, "docker://busybox")
	importImageAndFetchHash(t, ctx, "docker://busybox:latest")

	// 3. Try run/prepare with the image available in the store, should get $foundMsg.
	for _, cmd := range cmds {
		t.Logf("Running test %v", cmd)

		child, err := gexpect.Spawn(cmd)
		if err != nil {
			t.Fatalf("Cannot exec rkt: %v", err)
		}
		if err := expectWithOutput(child, foundMsg); err != nil {
			t.Fatalf("%q should be found", foundMsg)
		}
		if err := child.Wait(); err != nil {
			t.Fatalf("rkt didn't terminate correctly: %v", err)
		}
	}
}
