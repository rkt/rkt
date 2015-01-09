package util

import (
	"fmt"
	"os"
	"runtime"
	"syscall"
)

var setNsMap = map[string]uintptr{
	"386":   346,
	"amd64": 308,
	"arm":   374,
}

func SetNS(f *os.File, flags uintptr) error {
	if runtime.GOOS != "linux" {
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}

	trap, ok := setNsMap[runtime.GOARCH]
	if !ok {
		return fmt.Errorf("unsupported arch: %s", runtime.GOARCH)
	}
	_, _, err := syscall.RawSyscall(trap, f.Fd(), flags, 0)
	if err != 0 {
		return err
	}

	return nil
}

func WithNetNSPath(nspath string, f func(*os.File) error) error {
	// save a handle to current (host) network namespace
	thisNS, err := os.Open("/proc/self/ns/net")
	if err != nil {
		return fmt.Errorf("Failed to open /proc/self/ns/net: %v", err)
	}

	// switch to the container namespace
	ns, err := os.Open(nspath)
	if err != nil {
		return fmt.Errorf("Failed to open %v: %v", nspath, err)
	}

	if err = SetNS(ns, syscall.CLONE_NEWNET); err != nil {
		return fmt.Errorf("Error switching to ns %v: %v", nspath, err)
	}

	if err = f(thisNS); err != nil {
		return err
	}

	// switch back
	if err = SetNS(thisNS, syscall.CLONE_NEWNET); err != nil {
		return err
	}

	return nil
}
