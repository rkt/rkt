// Copyright 2014 CoreOS, Inc.
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
	"net/url"
	"os"
	"path/filepath"

	"github.com/appc/spec/discovery"
	"github.com/appc/spec/schema/types"
	"github.com/coreos/rocket/cas"
)

const (
	imgDir = "images"
)

var (
	cmdFetch = &Command{
		Name:    "fetch",
		Summary: "Fetch image(s) and store them in the local cache",
		Usage:   "IMAGE_URL...",
		Run:     runFetch,
	}
)

func init() {
	commands = append(commands, cmdFetch)
}

func fetchURL(img string, ds *cas.Store) (string, error) {
	rem := cas.NewRemote(img, []string{})
	err := ds.ReadIndex(rem)
	if err != nil && rem.Blob == "" {
		rem, err = rem.Download(*ds)
		if err != nil {
			return "", fmt.Errorf("downloading: %v\n", err)
		}
	}
	return rem.Blob, nil
}

// fetchImage will take an image as either a URL or a name string and import it
// into the store if found.
func fetchImage(img string, ds *cas.Store) (string, error) {
	// discover if it isn't a URL
	u, err := url.Parse(img)
	if err == nil && u.Scheme == "" {
		app, err := discovery.NewAppFromString(img)
		if globalFlags.Debug && err != nil {
			fmt.Printf("discovery: %s\n", err)
		}
		if err == nil {
			ep, err := discovery.DiscoverEndpoints(*app, true)
			if err != nil {
				return "", err
			}
			// TODO(philips): use all available mirrors
			if globalFlags.Debug {
				fmt.Printf("fetch: trying %v\n", ep.ACI)
			}
			img = ep.ACI[0]
			u, err = url.Parse(img)
		}
	}

	if err != nil { // download if it isn't a URL
		return "", fmt.Errorf("%s: not a valid URL or hash", img)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("%s: rkt only supports http or https URLs", img)
	}
	return fetchURL(img, ds)
}

func runFetch(args []string) (exit int) {
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "fetch: Must provide at least one image\n")
		return 1
	}
	root := filepath.Join(globalFlags.Dir, imgDir)
	if err := os.MkdirAll(root, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "fetch: error creating image directory: %v\n", err)
		return 1
	}

	ds := cas.NewStore(globalFlags.Dir)

	for _, img := range args {
		hash, err := fetchImage(img, ds)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			return 1
		}
		shortHash := types.ShortHash(hash)
		fmt.Println(shortHash)
	}

	return
}
