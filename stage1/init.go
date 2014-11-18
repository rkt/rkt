package main

// this implements /init of stage1/host_nspawn-systemd

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

const (
	SystemdNspawn = "systemd-nspawn"
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

	ex, err := exec.LookPath(SystemdNspawn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to locate %s executable: %v\n", SystemdNspawn, err)
		os.Exit(3)
	}

	args := []string{ex}

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
