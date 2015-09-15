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

//+build linux

package main

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/coreos/rkt/Godeps/_workspace/src/github.com/appc/spec/schema/types"
	"github.com/coreos/rkt/Godeps/_workspace/src/github.com/spf13/pflag"
	"github.com/coreos/rkt/store"
)

const (
	stage1ImageFlagName string = "stage1-image"
)

var (
	defaultStage1Image   string // either set by linker, or guessed in init()
	defaultStage1Name    string // set by linker
	defaultStage1Version string // set by linker

	flagStage1Image  string
)

func setDefaultStage1Image() {
	// if not set by linker, try discover the directory rkt is running
	// from, and assume the default stage1.aci is stored alongside it.
	if defaultStage1Image == "" {
		if exePath, err := os.Readlink("/proc/self/exe"); err == nil {
			defaultStage1Image = filepath.Join(filepath.Dir(exePath), "stage1.aci")
		}
	}
}

// addStage1ImageFlag adds a flag for specifying custom stage1 image
// path
func addStage1ImageFlag(flags *pflag.FlagSet) {
	flags.StringVar(&flagStage1Image, stage1ImageFlagName, defaultStage1Image, stage1ImageFlagHelp())
}

func stage1ImageFlagHelp() string {
	firstPart := "image to use as stage1. Local paths and http/https URLs are supported"
	return fmt.Sprintf(`%s. By default, rkt will look for a file called "stage1.aci" in the same directory as rkt itself`, firstPart)
}

// getStage1Hash will try to fetch stage1 from store if it is a
// default one. If that fails it will try to get via usual fetching
// from disk or network.
func getStage1Hash(s *store.Store, stage1ImagePath string) (*types.Hash, error) {
	fn := &finder{
		imageActionData: imageActionData{
			s: s,
		},
		local:    true,
		withDeps: false,
	}
	isDefault := stage1ImagePath == defaultStage1Image

	var (
		s1img *types.Hash
		err   error
	)
	// with a default stage1, first try to get the image from the store
	if isDefault {
		// we make sure we've built rkt with a clean git tree, otherwise we
		// don't know if something changed
		if !strings.HasSuffix(defaultStage1Version, "-dirty") {
			stage1AppName := fmt.Sprintf("%s:%s", defaultStage1Name, url.QueryEscape(defaultStage1Version))
			s1img, err = fn.findImage(stage1AppName, "", true)
		}
	}

	// we couldn't find a proper stage1 image in the store, fall back to using
	// a stage1 ACI file (slow)
	if s1img == nil {
		s1img, err = fn.findImage(stage1ImagePath, "", false)
		if err != nil {
			return nil, fmt.Errorf("error finding stage1 image %q: %v", stage1ImagePath, err)
		}
	}

	return s1img, nil
}
