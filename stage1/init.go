package main

// this implements /init of stage1/host_nspawn-systemd

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"github.com/coreos-inc/rkt/rkt"
)

const (
	// Path to systemd-nspawn binary within the stage1 rootfs
	nspawnBin = "/usr/bin/systemd-nspawn"
)

func main() {
	root := "."

	c, err := LoadContainer(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load container: %v\n", err)
		os.Exit(1)
	}

	if err = c.ContainerToSystemd(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to configure systemd: %v\n", err)
		os.Exit(2)
	}

	ex := filepath.Join(rkt.Stage1RootfsPath(c.Root), nspawnBin)
	if _, err := os.Stat(ex); err != nil {
		fmt.Fprintf(os.Stderr, "Failed locating nspawn: %v\n", err)
		os.Exit(3)
	}

	args := []string{
		ex,
		"--register", "false", // We cannot assume the host system is running a compatible systemd
	}

	nsargs, err := c.ContainerToNspawnArgs()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to generate nspawn args: %v\n", err)
		os.Exit(4)
	}

	args = append(args, nsargs...)

	env := os.Environ()

	if err := syscall.Exec(ex, args, env); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to execute nspawn: %v\n", err)
		os.Exit(5)
	}
}
