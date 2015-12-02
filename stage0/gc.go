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

package stage0

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"syscall"

	"github.com/coreos/rkt/Godeps/_workspace/src/github.com/appc/spec/schema/types"
)

// GC enters the pod by fork/exec()ing the stage1's /gc similar to /init.
// /gc can expect to have its CWD set to the pod root.
// stage1Path is the path of the stage1 rootfs
func GC(pdir string, uuid *types.UUID, stage1Path string, debug bool) error {
	err := unregisterPod(pdir, uuid)
	if err != nil {
		// Probably not worth abandoning the rest
		log.Printf("Warning: could not unregister pod with metadata service: %v", err)
	}

	ep, err := getStage1Entrypoint(pdir, gcEntrypoint)
	if err != nil {
		return fmt.Errorf("error determining 'gc' entrypoint: %v", err)
	}

	args := []string{filepath.Join(stage1Path, ep)}
	if debug {
		args = append(args, "--debug")
	}
	args = append(args, uuid.String())

	c := exec.Cmd{
		Path:   args[0],
		Args:   args,
		Stderr: os.Stderr,
		Dir:    pdir,
	}
	return c.Run()
}

type mount struct {
	id         int
	parentId   int
	mountPoint string
}

type mounts []*mount

// Less ensures that mounts are sorted in an order we can unmount; child before
// parent. This works by pushing all child mounts up to the front. Parent
// mounts are halted once they encounter a child. The actual sequential order
// of the mounts doesn't matter; only that children are in front of parents.
func (m mounts) Less(i, j int) bool { return m[i].id != m[j].parentId }
func (m mounts) Len() int           { return len(m) }
func (m mounts) Swap(i, j int)      { m[i], m[j] = m[j], m[i] }

// getMountsForPrefix parses mi (/proc/PID/mountinfo) and returns mounts for path prefix
func getMountsForPrefix(path string, mi io.Reader) (mounts, error) {
	var podMounts mounts
	sc := bufio.NewScanner(mi)
	for sc.Scan() {
		var (
			mountId    int
			parentId   int
			discard    string
			mountPoint string
		)

		_, err := fmt.Sscanf(sc.Text(), "%d %d %s %s %s",
			&mountId, &parentId, &discard, &discard, &mountPoint)
		if err != nil {
			return nil, err
		}

		if strings.HasPrefix(mountPoint, path) {
			mnt := &mount{
				id:         mountId,
				parentId:   parentId,
				mountPoint: mountPoint,
			}
			podMounts = append(podMounts, mnt)
		}
	}
	if sc.Err() != nil {
		return nil, fmt.Errorf("problem parsing mountinfo: %v", sc.Err())
	}

	sort.Sort(podMounts)
	return podMounts, nil
}

// ForceMountGC removes mounts from pods that couldn't be GCed cleanly.
func ForceMountGC(path, uuid string) error {
	mi, err := os.Open("/proc/self/mountinfo")
	if err != nil {
		return err
	}
	defer mi.Close()

	mnts, err := getMountsForPrefix(path, mi)
	if err != nil {
		return fmt.Errorf("error getting mounts for pod %s from mountinfo: %v", uuid, err)
	}

	for _, mnt := range mnts {
		if err := syscall.Unmount(mnt.mountPoint, 0); err != nil {
			return fmt.Errorf("could not unmount at %v: %v", mnt.mountPoint, err)
		}
	}
	return nil
}
