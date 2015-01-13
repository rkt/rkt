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

//+build linux

package main

import (
	"fmt"
	"os"
)

var (
	cmdList = &Command{
		Name:    "list",
		Summary: "List containers",
		Usage:   "",
		Run:     runList,
	}
	flagNoLegend bool
)

func init() {
	commands = append(commands, cmdList)
	cmdList.Flags.BoolVar(&flagNoLegend, "no-legend", false, "suppress a legend with the list")
}

func runList(args []string) (exit int) {
	if !flagNoLegend {
		fmt.Printf("UUID                                 STATE\n")
	}

	if err := walkContainers(includeContainersDir|includeGarbageDir, func(c *container) {
		state := "active"
		if c.isDeleting {
			state = "deleting"
		} else if c.isExited {
			state = "inactive"
		}

		fmt.Printf("%s %s\n", c.uuid, state)
	}); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get container handles: %v\n", err)
		return 1
	}

	return 0
}
