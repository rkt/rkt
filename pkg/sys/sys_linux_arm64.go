// +build linux,arm64

package sys

import "syscall"

const (
	SYS_SYNCFS = syscall.SYS_SYNCFS
)
