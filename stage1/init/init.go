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
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	"github.com/coreos/rocket/common"
	"github.com/coreos/rocket/networking"
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
	metadataSvc string
	privNet     bool
)

func init() {
	flag.BoolVar(&debug, "debug", false, "Run in debug mode")
	flag.BoolVar(&privNet, "private-net", false, "Setup private network (WIP!)")
	flag.StringVar(&metadataSvc, "metadata-svc", "", "Launch specified metadata svc")

	// this ensures that main runs only on main thread (thread group leader).
	// since namespace ops (unshare, setns) are done for a single thread, we
	// must ensure that the goroutine does not jump from OS thread to thread
	runtime.LockOSThread()
}

func stage1() int {
	root := "."
	c, err := LoadContainer(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load container: %v\n", err)
		return 1
	}

	mirrorLocalZoneInfo(c.Root)
	c.MetadataSvcURL = common.MetadataSvcPublicURL()

	if err = c.ContainerToSystemd(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to configure systemd: %v\n", err)
		return 2
	}

	args := []string{
		filepath.Join(common.Stage1RootfsPath(c.Root), interpBin),
		filepath.Join(common.Stage1RootfsPath(c.Root), nspawnBin),
		"--boot",              // Launch systemd in the container
		"--register", "false", // We cannot assume the host system is running systemd
	}

	if !debug {
		args = append(args, "--quiet") // silence most nspawn output (log_warning is currently not covered by this)
	}

	nsargs, err := c.ContainerToNspawnArgs()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to generate nspawn args: %v\n", err)
		return 4
	}
	args = append(args, nsargs...)

	// Arguments to systemd
	args = append(args, "--")
	args = append(args, "--default-standard-output=tty") // redirect all service logs straight to tty
	if !debug {
		args = append(args, "--log-target=null") // silence systemd output inside container
		args = append(args, "--show-status=0")   // silence systemd initialization status output
	}

	env := os.Environ()
	env = append(env, "LD_PRELOAD="+filepath.Join(common.Stage1RootfsPath(c.Root), "fakesdboot.so"))
	env = append(env, "LD_LIBRARY_PATH="+filepath.Join(common.Stage1RootfsPath(c.Root), "usr/lib"))

	if metadataSvc != "" {
		if err = launchMetadataSvc(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to launch metadata svc: %v\n", err)
			return 6
		}
	}

	if privNet {
		// careful not to make another local err variable.
		// cmd.Run sets the one from parent scope
		var n *networking.Networking
		n, err = networking.Setup(root, c.Manifest.UUID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to setup network: %v\n", err)
			return 6
		}
		defer n.Teardown()

		if err = n.EnterContNS(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to switch to container netns: %v\n", err)
			return 6
		}

		if err = registerContainer(c, n.MetadataIP); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to register container: %v\n", err)
			return 6
		}
		defer unregisterContainer(c)

		cmd := exec.Cmd{
			Path:   args[0],
			Args:   args,
			Stdin:  os.Stdin,
			Stdout: os.Stdout,
			Stderr: os.Stderr,
			Env:    env,
		}
		err = cmd.Run()
	} else {
		err = syscall.Exec(args[0], args, env)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to execute nspawn: %v\n", err)
		return 5
	}

	return 0
}

func launchMetadataSvc() error {
	log.Print("Launching metadatasvc: ", metadataSvc)

	// use socket activation protocol to avoid race-condition of
	// service becoming ready
	// TODO(eyakubovich): remove hard-coded port
	l, err := net.ListenTCP("tcp4", &net.TCPAddr{Port: common.MetadataSvcPrvPort})
	if err != nil {
		if err.(*net.OpError).Err.(*os.SyscallError).Err == syscall.EADDRINUSE {
			// assume metadatasvc is already running
			return nil
		}
		return err
	}

	defer l.Close()

	lf, err := l.File()
	if err != nil {
		return err
	}

	// parse metadataSvc into exe and args
	args := strings.Split(metadataSvc, " ")
	cmd := exec.Cmd{
		Path:       args[0],
		Args:       args,
		Env:        append(os.Environ(), "LISTEN_FDS=1"),
		ExtraFiles: []*os.File{lf},
		Stdout:     os.Stdout,
		Stderr:     os.Stderr,
	}
	return cmd.Start()
}

func main() {
	flag.Parse()
	// move code into stage1() helper so defered fns get run
	os.Exit(stage1())
}
