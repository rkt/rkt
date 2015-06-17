// Copyright 2015 The appc Authors
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
	"runtime"
	"strings"

	"github.com/coreos/rkt/Godeps/_workspace/src/github.com/appc/spec/discovery"
)

var (
	cmdDiscover = &Command{
		Name:        "discover",
		Description: "Discover the download URLs for an app",
		Summary:     "Discover the download URLs for one or more app container images",
		Usage:       "APP...",
		Run:         runDiscover,
	}
)

func init() {
	cmdDiscover.Flags.BoolVar(&transportFlags.Insecure, "insecure", false,
		"Allow insecure non-TLS downloads over http")
}

func runDiscover(args []string) (exit int) {
	if len(args) < 1 {
		stderr("discover: at least one name required")
	}

	for _, name := range args {
		app, err := discovery.NewAppFromString(name)
		if app.Labels["os"] == "" {
			app.Labels["os"] = runtime.GOOS
		}
		if app.Labels["arch"] == "" {
			app.Labels["arch"] = runtime.GOARCH
		}
		if err != nil {
			stderr("%s: %s", name, err)
			return 1
		}
		eps, attempts, err := discovery.DiscoverEndpoints(*app, transportFlags.Insecure)
		if err != nil {
			stderr("error fetching %s: %s", name, err)
			return 1
		}
		for _, a := range attempts {
			fmt.Printf("discover walk: prefix: %s error: %v\n", a.Prefix, a.Error)
		}
		for _, aciEndpoint := range eps.ACIEndpoints {
			fmt.Printf("ACI: %s, ASC: %s\n", aciEndpoint.ACI, aciEndpoint.ASC)
		}
		if len(eps.Keys) > 0 {
			fmt.Println("Keys: " + strings.Join(eps.Keys, ","))
		}
	}

	return
}
