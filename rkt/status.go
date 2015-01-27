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

//+build linux

package main

import (
	"fmt"
	"os"

	"github.com/appc/spec/schema/types"
)

var (
	cmdStatus = &Command{
		Name:    cmdStatusName,
		Summary: "Check the status of a rkt container",
		Usage:   "[--wait] UUID",
		Run:     runStatus,
	}
	flagWait bool
)

const (
	statusDir     = "stage1/rootfs/rkt/status"
	cmdStatusName = "status"
)

func init() {
	commands = append(commands, cmdStatus)
	cmdStatus.Flags.BoolVar(&flagWait, "wait", false, "toggle waiting for the container to exit if running")
}

func runStatus(args []string) (exit int) {
	if len(args) != 1 {
		printCommandUsageByName(cmdStatusName)
		return 1
	}

	containerUUID, err := types.NewUUID(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid UUID: %v\n", err)
		return 1
	}

	ch, err := getContainer(containerUUID.String())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to get container handle: %v\n", err)
		return 1
	}
	defer ch.Close()

	if flagWait {
		if err := ch.waitExited(); err != nil {
			fmt.Fprintf(os.Stderr, "Unable to wait for container: %v\n", err)
			return 1
		}
	}

	if err = printStatus(ch); err != nil {
		fmt.Fprintf(os.Stderr, "Unable to print status: %v\n", err)
		return 1
	}

	return 0
}

// printStatus prints the container's pid and per-app status codes
func printStatus(ch *container) error {
	pid, err := ch.getPID()
	if err != nil {
		return err
	}

	stats, err := ch.getStatuses()
	if err != nil {
		return err
	}

	fmt.Printf("pid=%d\nexited=%t\n", pid, ch.isExited)
	for app, stat := range stats {
		fmt.Printf("%s=%d\n", app, stat)
	}
	return nil
}
