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

// this implements /init of stage1/nspawn+systemd

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"

	"github.com/coreos/rkt/Godeps/_workspace/src/github.com/appc/spec/schema/types"

	"github.com/coreos/rkt/common"
	"github.com/coreos/rkt/networking"
	"github.com/coreos/rkt/pkg/sys"
)

const (
	// Path to systemd-nspawn binary within the stage1 rootfs
	nspawnBin = "/usr/bin/systemd-nspawn"
	// Path to the interpreter within the stage1 rootfs
	interpBin = "/usr/lib/ld-linux-x86-64.so.2"
	// Path to the localtime file/symlink in host
	localtimePath = "/etc/localtime"
)

// mirrorLocalZoneInfo tries to reproduce the /etc/localtime target in stage1/ to satisfy systemd-nspawn
func mirrorLocalZoneInfo(root string) {
	zif, err := os.Readlink(localtimePath)
	if err != nil {
		return
	}

	// On some systems /etc/localtime is a relative symlink, make it absolute
	if !filepath.IsAbs(zif) {
		zif = filepath.Join(filepath.Dir(localtimePath), zif)
		zif = filepath.Clean(zif)
	}

	src, err := os.Open(zif)
	if err != nil {
		return
	}
	defer src.Close()

	destp := filepath.Join(common.Stage1RootfsPath(root), zif)

	if err = os.MkdirAll(filepath.Dir(destp), 0755); err != nil {
		return
	}

	dest, err := os.OpenFile(destp, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer dest.Close()

	_, _ = io.Copy(dest, src)
}

var (
	debug       bool
	privNet     bool
	interactive bool
)

func init() {
	flag.BoolVar(&debug, "debug", false, "Run in debug mode")
	flag.BoolVar(&privNet, "private-net", false, "Setup private network (WIP!)")
	flag.BoolVar(&interactive, "interactive", false, "The pod is interactive")

	// this ensures that main runs only on main thread (thread group leader).
	// since namespace ops (unshare, setns) are done for a single thread, we
	// must ensure that the goroutine does not jump from OS thread to thread
	runtime.LockOSThread()
}

// getArgsEnv returns the nspawn args and env according to the usr used
func getArgsEnv(p *Pod, debug bool) ([]string, []string, error) {
	args := []string{}
	env := os.Environ()

	flavor, err := os.Readlink(filepath.Join(common.Stage1RootfsPath(p.Root), "flavor"))
	if err != nil {
		return nil, nil, fmt.Errorf("unable to determine stage1 flavor: %v", err)
	}

	switch flavor {
	case "coreos":
		// when running the coreos-derived stage1 with unpatched systemd-nspawn we need some ld-linux hackery
		args = append(args, filepath.Join(common.Stage1RootfsPath(p.Root), interpBin))
		args = append(args, filepath.Join(common.Stage1RootfsPath(p.Root), nspawnBin))
		args = append(args, "--boot")              // Launch systemd in the pod
		args = append(args, "--register", "false") // We cannot assume the host system is running systemd
		// TODO(vc): we should leave registration enabled if systemd is running on the host,
		// but it needs to be sufficiently new systemd or registration will fail.

		env = append(env, "LD_PRELOAD="+filepath.Join(common.Stage1RootfsPath(p.Root), "fakesdboot.so"))
		env = append(env, "LD_LIBRARY_PATH="+filepath.Join(common.Stage1RootfsPath(p.Root), "usr/lib"))

	case "src":
		args = append(args, filepath.Join(common.Stage1RootfsPath(p.Root), nspawnBin))
		args = append(args, "--boot") // Launch systemd in the pod

	default:
		return nil, nil, fmt.Errorf("unrecognized stage1 flavor: %q", flavor)
	}

	if !debug {
		args = append(args, "--quiet") // silence most nspawn output (log_warning is currently not covered by this)
	}

	nsargs, err := p.PodToNspawnArgs()
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to generate nspawn args: %v", err)
	}
	args = append(args, nsargs...)

	// Arguments to systemd
	args = append(args, "--")
	args = append(args, "--default-standard-output=tty") // redirect all service logs straight to tty
	if !debug {
		args = append(args, "--log-target=null") // silence systemd output inside pod
		args = append(args, "--show-status=0")   // silence systemd initialization status output
	}

	return args, env, nil
}

func withClearedCloExec(lfd int, f func() error) error {
	err := sys.CloseOnExec(lfd, false)
	if err != nil {
		return err
	}
	defer sys.CloseOnExec(lfd, true)

	return f()
}

func stage1() int {
	uuid, err := types.NewUUID(flag.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, "UUID is missing or malformed")
		return 1
	}

	root := "."
	p, err := LoadPod(root, uuid)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load pod: %v\n", err)
		return 1
	}

	// set close-on-exec flag on RKT_LOCK_FD so it gets correctly closed when invoking
	// network plugins
	lfd, err := common.GetRktLockFD()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get rkt lock fd: %v\n", err)
		return 1
	}

	if err := sys.CloseOnExec(lfd, true); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to set FD_CLOEXEC on rkt lock: %v\n", err)
		return 1
	}

	mirrorLocalZoneInfo(p.Root)

	if privNet {
		n, err := networking.Setup(root, p.UUID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to setup network: %v\n", err)
			return 6
		}
		defer n.Teardown()

		p.MetadataServiceURL = common.MetadataServicePublicURL(n.HostIP)

		if err = n.EnterPodNS(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to switch to pod netns: %v\n", err)
			return 6
		}

		if err = registerPod(p, n.MetadataIP); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to register pod: %v\n", err)
			return 6
		}
		defer unregisterPod(p)
	}

	if err = p.PodToSystemd(interactive); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to configure systemd: %v\n", err)
		return 2
	}

	args, env, err := getArgsEnv(p, debug)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get execution parameters: %v\n", err)
		return 3
	}

	var execFn func() error

	if privNet {
		cmd := exec.Cmd{
			Path:   args[0],
			Args:   args,
			Stdin:  os.Stdin,
			Stdout: os.Stdout,
			Stderr: os.Stderr,
			Env:    env,
		}
		execFn = cmd.Run
	} else {
		execFn = func() error {
			return syscall.Exec(args[0], args, env)
		}
	}

	err = withClearedCloExec(lfd, execFn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to execute nspawn: %v\n", err)
		return 5
	}

	return 0
}

func main() {
	flag.Parse()
	// move code into stage1() helper so defered fns get run
	os.Exit(stage1())
}
