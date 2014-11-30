package main

// this implements /init of stage1/host_nspawn-systemd

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"github.com/coreos/rocket/rkt"
)

const (
	// Path to systemd-nspawn binary within the stage1 rootfs
	nspawnBin = "/usr/bin/systemd-nspawn"
)

func main() {
	root := "."
	debug := len(os.Args) > 1 && os.Args[1] == "debug"

	c, err := LoadContainer(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load container: %v\n", err)
		os.Exit(1)
	}

	if err = c.ContainerToSystemd(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to configure systemd: %v\n", err)
		os.Exit(2)
	}

	// TODO(philips): compile a static version of systemd-nspawn with this
	// stupidity patched out
	_, err = os.Stat("/run/systemd/system")
	if os.IsNotExist(err) {
		os.MkdirAll("/run/systemd/system", 0755)
	}

	ex := filepath.Join(rkt.Stage1RootfsPath(c.Root), nspawnBin)
	if _, err := os.Stat(ex); err != nil {
		fmt.Fprintf(os.Stderr, "Failed locating nspawn: %v\n", err)
		os.Exit(3)
	}

	args := []string{
		ex,
		"--boot",              // Launch systemd in the container
		"--register", "false", // We cannot assume the host system is running a compatible systemd
	}

	if !debug {
		args = append(args, "--quiet") // silence most nspawn output (log_warning is currently not covered by this)
	}

	nsargs, err := c.ContainerToNspawnArgs()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to generate nspawn args: %v\n", err)
		os.Exit(4)
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

	if err := syscall.Exec(ex, args, env); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to execute nspawn: %v\n", err)
		os.Exit(5)
	}
}
