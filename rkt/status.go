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
	"io/ioutil"
	"os"
	"path/filepath"
	"syscall"

	"github.com/appc/spec/schema/types"
	"github.com/coreos/rocket/pkg/lock"
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
	statusDir     = "stage1/rkt/status"
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

	l, exited, err := getContainerLockAndState(containerUUID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to access container: %v\n", err)
		return 1
	}
	defer l.Close()

	// There's a window between opening the container directory and lock acquisition where gc rename could occur,
	// perform all subsequent opens relative to the opened container directory lock fd
	cfd, err := l.Fd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to get lock fd: %v\n", err)
		return 1
	}

	if err = printStatusAt(cfd, exited); err != nil {
		fmt.Fprintf(os.Stderr, "Unable to print status: %v\n", err)
		return 1
	}

	return 0
}

// getContainerLockAndState opens the container directory in the form of a lock.DirLock,
// returning the lock and wether the container has already exited or not.
func getContainerLockAndState(containerUUID *types.UUID) (l *lock.DirLock, isExited bool, err error) {
	cid := containerUUID.String()
	isGarbage := false

	cp := filepath.Join(containersDir(), cid)
	l, err = lock.NewLock(cp)
	if err == lock.ErrNotExist {
		// Fallback to garbage/$cid if containers/$cid is missing, "rkt gc" renames exited containers to garbage/$cid.
		isGarbage = true
		cp = filepath.Join(garbageDir(), cid)
		l, err = lock.NewLock(cp)
	}

	if err != nil {
		if err == lock.ErrNotExist {
			err = fmt.Errorf("container %v not found", cid)
		} else {
			err = fmt.Errorf("error opening lock: %v", err)
		}
		return
	}

	isExited = true
	if flagWait && !isGarbage {
		err = l.SharedLock()
	} else {
		err = l.TrySharedLock()
		if err == lock.ErrLocked {
			if isGarbage {
				// Container is exited and being deleted, we can't reliably query its status, it's effectively gone.
				err = fmt.Errorf("unable to query status: %q is being removed", cid)
				return
			}
			isExited = false
			err = nil
		}
	}

	if err != nil {
		err = fmt.Errorf("error acquiring lock: %v", err)
	}

	return
}

// printStatusAt prints the container's pid and per-app status codes
func printStatusAt(cdirfd int, exited bool) error {
	pid, err := getIntFromFileAt(cdirfd, "pid")
	if err != nil {
		return err
	}

	stats, err := getStatusesAt(cdirfd)
	if err != nil {
		return err
	}

	fmt.Printf("pid=%d\nexited=%t\n", pid, exited)
	for app, stat := range stats {
		fmt.Printf("%s=%d\n", app, stat)
	}
	return nil
}

// getStatusesAt returns a map of imageId:status codes for the given container
func getStatusesAt(cdirfd int) (map[string]int, error) {
	sdirfd, err := syscall.Openat(cdirfd, statusDir, syscall.O_RDONLY|syscall.O_DIRECTORY, 0)
	if err != nil {
		return nil, fmt.Errorf("unable to open status directory: %v", err)
	}
	sdir := os.NewFile(uintptr(sdirfd), statusDir)
	defer sdir.Close()

	ls, err := sdir.Readdirnames(0)
	if err != nil {
		return nil, fmt.Errorf("unable to read status directory: %v", err)
	}

	stats := make(map[string]int)
	for _, name := range ls {
		s, err := getIntFromFileAt(sdirfd, name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to get status of app %q: %v\n", name, err)
			continue
		}
		stats[name] = s
	}

	return stats, err
}

// getIntFromFileAt reads an integer string from the named file
func getIntFromFileAt(dirfd int, path string) (i int, err error) {
	fd, err := syscall.Openat(dirfd, path, syscall.O_RDONLY, 0)
	if err != nil {
		return
	}
	f := os.NewFile(uintptr(fd), path)
	defer f.Close()

	buf, err := ioutil.ReadAll(f)
	if err != nil {
		return
	}

	_, err = fmt.Sscanf(string(buf), "%d", &i)

	return
}
