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
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/coreos/rkt/Godeps/_workspace/src/github.com/appc/spec/schema/types"
	"github.com/coreos/rkt/Godeps/_workspace/src/github.com/coreos/go-systemd/unit"
)

type addIsolatorFunc func(opts []*unit.UnitOption, limit string) ([]*unit.UnitOption, error)

var (
	isolatorFuncs = map[string]addIsolatorFunc{
		"cpu":    addCpuLimit,
		"memory": addMemoryLimit,
	}
	cgroupControllerRWFiles = map[string][]string{
		"memory": []string{"memory.limit_in_bytes"},
		"cpu":    []string{"cpu.cfs_quota_us"},
	}
)

func addCpuLimit(opts []*unit.UnitOption, limit string) ([]*unit.UnitOption, error) {
	milliCores, err := strconv.Atoi(limit)
	if err != nil {
		return nil, err
	}
	quota := strconv.Itoa(milliCores/10) + "%"
	opts = append(opts, newUnitOption("Service", "CPUQuota", quota))
	return opts, nil
}

func addMemoryLimit(opts []*unit.UnitOption, limit string) ([]*unit.UnitOption, error) {
	opts = append(opts, newUnitOption("Service", "MemoryLimit", limit))
	return opts, nil
}

func maybeAddIsolator(opts []*unit.UnitOption, isolator string, limit string) ([]*unit.UnitOption, error) {
	var err error
	if isIsolatorSupported(isolator) {
		opts, err = isolatorFuncs[isolator](opts, limit)
		if err != nil {
			return nil, err
		}
	} else {
		fmt.Fprintf(os.Stderr, "warning: resource/%s isolator set but support disabled in the kernel, skipping\n", isolator)
	}
	return opts, nil
}

func isIsolatorSupported(isolator string) bool {
	if files, ok := cgroupControllerRWFiles[isolator]; ok {
		for _, f := range files {
			isolatorPath := filepath.Join("/sys/fs/cgroup/", isolator, f)
			if _, err := os.Stat(isolatorPath); os.IsNotExist(err) {
				return false
			}
		}
		return true
	}
	return false
}

func parseCgroups(f io.Reader) (map[int][]string, error) {
	sc := bufio.NewScanner(f)

	// skip first line since it is a comment
	sc.Scan()

	cgroups := make(map[int][]string)
	for sc.Scan() {
		var controller string
		var hierarchy int
		var num int
		var enabled int
		fmt.Sscanf(sc.Text(), "%s %d %d %d", &controller, &hierarchy, &num, &enabled)

		if enabled == 1 {
			if _, ok := cgroups[hierarchy]; !ok {
				cgroups[hierarchy] = []string{controller}
			} else {
				cgroups[hierarchy] = append(cgroups[hierarchy], controller)
			}
		}
	}

	if err := sc.Err(); err != nil {
		return nil, err
	}

	return cgroups, nil
}

func getControllers(cgroups map[int][]string) []string {
	var controllers []string
	for _, cs := range cgroups {
		controllers = append(controllers, strings.Join(cs, ","))
	}

	return controllers
}

func getControllerSymlinks(cgroups map[int][]string) map[string]string {
	symlinks := make(map[string]string)

	for _, cs := range cgroups {
		if len(cs) > 1 {
			tgt := strings.Join(cs, ",")
			for _, ln := range cs {
				symlinks[ln] = tgt
			}
		}
	}

	return symlinks
}

func getControllerRWFiles(controller string) []string {
	parts := strings.Split(controller, ",")
	for _, p := range parts {
		if files, ok := cgroupControllerRWFiles[p]; ok {
			// cgroup.procs always needs to be RW for allowing systemd to add
			// processes to the controller
			files = append(files, "cgroup.procs")
			return files
		}
	}

	return nil
}

func getOwnCgroupPath(controller string) (string, error) {
	selfCgroupPath := "/proc/self/cgroup"
	cg, err := os.Open(selfCgroupPath)
	if err != nil {
		return "", fmt.Errorf("error opening /proc/self/cgroup: %v", err)
	}
	defer cg.Close()

	s := bufio.NewScanner(cg)
	for s.Scan() {
		parts := strings.Split(s.Text(), ":")
		if parts[1] == controller {
			return parts[2], nil
		}
	}

	return "", fmt.Errorf("controller %q not found", controller)
}

// createCgroups mounts the cgroup controllers hierarchy for the container but
// leaves the subcgroup for each app read-write so the systemd inside stage1
// can apply isolators to them
func createCgroups(root string, subcgroup string, appHashes []types.Hash) error {
	cgroupsFile, err := os.Open("/proc/cgroups")
	if err != nil {
		return err
	}
	defer cgroupsFile.Close()

	cgroups, err := parseCgroups(cgroupsFile)
	if err != nil {
		return fmt.Errorf("error parsing /proc/cgroups: %v", err)
	}

	controllers := getControllers(cgroups)

	var flags uintptr

	// 1. Mount /sys read-only
	sys := filepath.Join(root, "/sys")
	if err := os.MkdirAll(sys, 0700); err != nil {
		return err
	}
	flags = syscall.MS_RDONLY |
		syscall.MS_NOSUID |
		syscall.MS_NOEXEC |
		syscall.MS_NODEV
	if err := syscall.Mount("sysfs", sys, "sysfs", flags, ""); err != nil {
		return fmt.Errorf("error mounting %q: %v", sys, err)
	}

	// 2. Mount /sys/fs/cgroup
	cgroupTmpfs := filepath.Join(root, "/sys/fs/cgroup")
	if err := os.MkdirAll(cgroupTmpfs, 0700); err != nil {
		return err
	}

	flags = syscall.MS_NOSUID |
		syscall.MS_NOEXEC |
		syscall.MS_NODEV |
		syscall.MS_STRICTATIME
	if err := syscall.Mount("tmpfs", cgroupTmpfs, "tmpfs", flags, "mode=755"); err != nil {
		return fmt.Errorf("error mounting %q: %v", cgroupTmpfs, err)
	}

	// 3. Mount controllers
	for _, c := range controllers {
		// 3a. Mount controller
		cPath := filepath.Join(root, "/sys/fs/cgroup", c)
		if err := os.MkdirAll(cPath, 0700); err != nil {
			return err
		}

		flags = syscall.MS_NOSUID |
			syscall.MS_NOEXEC |
			syscall.MS_NODEV
		if err := syscall.Mount("cgroup", cPath, "cgroup", flags, c); err != nil {
			return fmt.Errorf("error mounting %q: %v", cPath, err)
		}

		// 3b. Check if we're running from a unit to know which subcgroup
		// directories to mount read-write
		subcgroupPath := filepath.Join(cPath, subcgroup)

		// 3c. Create cgroup directories and mount the files we need over
		// themselves so they stay read-write
		for _, a := range appHashes {
			serviceName := ServiceUnitName(a)
			appCgroup := filepath.Join(subcgroupPath, serviceName)
			if err := os.MkdirAll(appCgroup, 0755); err != nil {
				return err
			}
			for _, f := range getControllerRWFiles(c) {
				cgroupFilePath := filepath.Join(appCgroup, f)
				// the file may not be there if kernel doesn't support the
				// feature, skip it in that case
				if _, err := os.Stat(cgroupFilePath); os.IsNotExist(err) {
					continue
				}
				if err := syscall.Mount(cgroupFilePath, cgroupFilePath, "", syscall.MS_BIND, ""); err != nil {
					return fmt.Errorf("error bind mounting %q: %v", cgroupFilePath, err)
				}
			}
		}

		// 3d. Re-mount controller read-only to prevent the container modifying host controllers
		flags = syscall.MS_BIND |
			syscall.MS_REMOUNT |
			syscall.MS_NOSUID |
			syscall.MS_NOEXEC |
			syscall.MS_NODEV |
			syscall.MS_RDONLY
		if err := syscall.Mount(cPath, cPath, "", flags, ""); err != nil {
			return fmt.Errorf("error remounting RO %q: %v", cPath, err)
		}
	}

	// 4. Create symlinks for combined controllers
	symlinks := getControllerSymlinks(cgroups)
	for ln, tgt := range symlinks {
		lnPath := filepath.Join(cgroupTmpfs, ln)
		if err := os.Symlink(tgt, lnPath); err != nil {
			return fmt.Errorf("error creating symlink: %v", err)
		}
	}

	// 5. Create systemd cgroup directory
	// We're letting systemd-nspawn create the systemd cgroup but later we're
	// remounting /sys/fs/cgroup read-only so we create the directory here.
	if err := os.MkdirAll(filepath.Join(cgroupTmpfs, "systemd"), 0700); err != nil {
		return err
	}

	// 6. Bind-mount cgroup filesystem read-only
	flags = syscall.MS_BIND |
		syscall.MS_REMOUNT |
		syscall.MS_NOSUID |
		syscall.MS_NOEXEC |
		syscall.MS_NODEV |
		syscall.MS_RDONLY
	if err := syscall.Mount(cgroupTmpfs, cgroupTmpfs, "", flags, ""); err != nil {
		return fmt.Errorf("error remounting RO %q: %v", cgroupTmpfs, err)
	}

	return nil
}
