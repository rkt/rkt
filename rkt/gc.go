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
	"io/ioutil"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/coreos/rocket/pkg/lock"
)

const (
	defaultGracePeriod = 30 * time.Minute
)

var (
	flagGracePeriod time.Duration
	cmdGC           = &Command{
		Name:    "gc",
		Summary: "Garbage-collect rkt containers no longer in use",
		Usage:   "[--grace-period=duration]",
		Run:     runGC,
	}
)

func init() {
	commands = append(commands, cmdGC)
	cmdGC.Flags.DurationVar(&flagGracePeriod, "grace-period", defaultGracePeriod, "duration to wait before discarding inactive containers from garbage")
}

func runGC(args []string) (exit int) {
	if err := os.MkdirAll(garbageDir(), 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Unable to create garbage dir: %v\n", err)
		return 1
	}

	cs, err := getContainers()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to get containers list: %v\n", err)
		return 1
	}
	for _, c := range cs {
		cp := filepath.Join(containersDir(), c)
		// XXX: Note this isn't serializing anything, the rename is atomic and things like `rkt status` may occur simultaneously.
		//      Care has been taken in the implementation of `rkt status` to be immune to the concurrent rename.
		l, err := lock.TrySharedLock(cp)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to open lock, ignoring %q: %v\n", c, err)
			continue
		}

		fmt.Printf("Moving container %q to garbage\n", c)
		err = os.Rename(cp, filepath.Join(garbageDir(), c))
		if err != nil {
			fmt.Println(err)
		}
		l.Close()
	}

	// clean up anything old in the garbage dir
	err = emptyGarbage(flagGracePeriod)
	if err != nil {
		fmt.Println(err)
		return 1
	}

	return
}

// getContainers returns a slice representing the containers in the given rocket directory
func getContainers() ([]string, error) {
	cdir := containersDir()
	ls, err := ioutil.ReadDir(cdir)
	if err != nil {
		return nil, fmt.Errorf("cannot read containers directory: %v", err)
	}
	var cs []string
	for _, dir := range ls {
		if !dir.IsDir() {
			fmt.Fprintf(os.Stderr, "Unrecognized file: %q, ignoring\n", dir)
			continue
		}
		cs = append(cs, dir.Name())
	}
	return cs, nil
}

// emptyGarbage discards sufficiently aged containers from garbageDir()
func emptyGarbage(gracePeriod time.Duration) error {
	g := garbageDir()

	ls, err := ioutil.ReadDir(g)
	if err != nil {
		return err
	}

	for _, dir := range ls {
		gp := filepath.Join(g, dir.Name())
		st := &syscall.Stat_t{}
		err := syscall.Lstat(gp, st)
		if err != nil {
			if err != syscall.ENOENT {
				fmt.Fprintf(os.Stderr, "Unable to stat %q, ignoring: %v\n", gp, err)
			}
			continue
		}

		expiration := time.Unix(st.Ctim.Unix()).Add(gracePeriod)
		if time.Now().After(expiration) {
			l, err := lock.ExclusiveLock(gp)
			if err != nil {
				continue
			}
			fmt.Printf("Garbage collecting container %q\n", dir.Name())
			if err = os.RemoveAll(gp); err != nil {
				fmt.Fprintf(os.Stderr, "Unable to remove container %q: %v\n", dir.Name(), err)
			}
			l.Close()
		}
	}
	return nil
}
