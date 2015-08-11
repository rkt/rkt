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
	"os"

	"github.com/coreos/rkt/Godeps/_workspace/src/github.com/appc/spec/schema/types"
	"github.com/coreos/rkt/Godeps/_workspace/src/github.com/spf13/cobra"
)

var (
	cmdRm = &cobra.Command{
		Use:   "rm [--uuid-file=FILE] UUID",
		Short: "Remove all files and resources associated with an exited pod",
		Run:   runWrapper(runRm),
	}
	flagUUIDFile string
)

func init() {
	cmdRkt.AddCommand(cmdRm)
	cmdRm.Flags().StringVar(&flagUUIDFile, "uuid-file", "", "read pod UUID from file instead of argument")
}

func runRm(cmd *cobra.Command, args []string) (exit int) {
	var podUUID *types.UUID
	var err error

	switch {
	case len(args) == 0 && flagUUIDFile != "":
		podUUID, err = readUUIDFromFile(flagUUIDFile)
		if err != nil {
			stderr("Unable to read UUID from file: %v", err)
			return 1
		}

	case len(args) == 1 && flagUUIDFile == "":
		podUUID, err = resolveUUID(args[0])
		if err != nil {
			stderr("Unable to resolve UUID: %v", err)
			return 1
		}
	default:
		cmd.Usage()
		return 1
	}

	p, err := getPod(podUUID)
	if err != nil {
		stderr("Cannot get pod: %v", err)
		return 1
	}

	return removePod(p)
}

func removePod(p *pod) int {
	switch {
	case p.isRunning():
		stderr("Pod is currently running")
		return 1

	case p.isEmbryo, p.isPreparing:
		stderr("Pod is currently being prepared")
		return 1

	case p.isExitedDeleting, p.isDeleting:
		stderr("Pod is currently being deleted")
		return 1

	case p.isAbortedPrepare:
		stderr("Moving failed prepare %q to garbage", p.uuid)
		if err := p.xToGarbage(); err != nil && err != os.ErrNotExist {
			stderr("Rename error: %v", err)
			return 1
		}

	case p.isPrepared:
		stderr("Moving expired prepared pod %q to garbage", p.uuid)
		if err := p.xToGarbage(); err != nil && err != os.ErrNotExist {
			stderr("Rename error: %v", err)
			return 1
		}

	case p.isExited:
		if err := p.xToExitedGarbage(); err != nil && err != os.ErrNotExist {
			stderr("Rename error: %v", err)
			return 1
		}

	case p.isExitedGarbage, p.isGarbage:
	}

	if err := p.ExclusiveLock(); err != nil {
		stderr("Unable to acquire exclusive lock: %v", err)
		return 1
	}

	deletePod(p)

	return 0
}
