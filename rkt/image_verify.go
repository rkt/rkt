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
	"os"

	"github.com/hashicorp/errwrap"
	"github.com/rkt/rkt/store/imagestore"
	"github.com/rkt/rkt/store/treestore"
	"github.com/spf13/cobra"
)

var (
	cmdImageVerify = &cobra.Command{
		Use:   "verify IMAGE...",
		Short: "Verify one or more rendered images in the local store",
		Long: `Verify is able to check that, based on the stored hash value, a rendered image on disk has not changed.

This command may be used if the user suspects disk corruption might have damaged the rkt store.`,
		Run: runWrapper(runVerifyImage),
	}
)

func init() {
	cmdImage.AddCommand(cmdImageVerify)
}

func runVerifyImage(cmd *cobra.Command, args []string) int {
	if len(args) < 1 {
		stderr.Print("must provide at least one image ID")
		return 254
	}

	s, err := imagestore.NewStore(storeDir())
	if err != nil {
		stderr.PrintE("cannot open store", err)
		return 254
	}

	ts, err := treestore.NewStore(treeStoreDir(), s)
	if err != nil {
		stderr.PrintE("cannot open treestore", err)
		return 254
	}

	for _, img := range args {
		key, err := getStoreKeyFromAppOrHash(s, img)
		if err != nil {
			stderr.Printf("unable to resolve store key for image %s: %v", img, err)
			return 254
		}
		id, err := ts.GetID(key)
		if err != nil {
			stderr.Printf("unable to get treestoreID for image %s: %v", img, err)
			return 254
		}
		_, err = ts.Check(id)
		if isNotRenderedErr(err) {
			stdout.Printf("image %q not rendered; no verification needed", img)
			continue
		}
		if err != nil {
			stdout.Printf("tree cache verification failed for image %s: %v;  rebuilding...", img, err)
			_, _, err = ts.Render(key, true)
			if err != nil {
				stderr.Printf("unable to repair cache for image %s: %v", img, err)
				return 254
			}
		} else {
			stdout.Printf("successfully verified checksum for image: %q (%q)", img, key)
		}
	}
	return 0
}

func isNotRenderedErr(err error) bool {
	containsIsNotExist := false
	containsReadHashErr := false
	errwrap.Walk(err, func(e error) {
		if os.IsNotExist(e) {
			containsIsNotExist = true
		}
		if e == treestore.ErrReadHashfile {
			containsReadHashErr = true
		}
	})
	return containsIsNotExist && containsReadHashErr
}
