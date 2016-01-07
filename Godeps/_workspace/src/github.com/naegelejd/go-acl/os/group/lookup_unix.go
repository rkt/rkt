// Derived from os/user/lookup_unix.go in the Go source code
// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
// - Joseph Naegele 2015

// +build darwin dragonfly freebsd !android,linux netbsd openbsd solaris
// +build cgo

package group

import (
	"fmt"
	"runtime"
	"strconv"
	"syscall"
	"unsafe"
)

/*
#include <unistd.h>
#include <grp.h>
#include <stdlib.h>

static int mygetgrgid_r(int uid, struct group *grp,
	char *buf, size_t buflen, struct group **result) {
    return getgrgid_r(uid, grp, buf, buflen, result);
}
*/
import "C"

func current() (*Group, error) {
	return lookupUnix(syscall.Getgid(), "", false)
}

func lookup(groupname string) (*Group, error) {
	return lookupUnix(-1, groupname, true)
}

func lookupId(gid string) (*Group, error) {
	id, err := strconv.Atoi(gid)
	if err != nil {
		return nil, err
	}
	return lookupUnix(id, "", false)
}

func lookupUnix(gid int, groupname string, lookupByName bool) (*Group, error) {
	var grp C.struct_group
	var result *C.struct_group

	var bufSize C.long
	if runtime.GOOS == "dragonfly" || runtime.GOOS == "freebsd" {
		// DragonFly and FreeBSD do not have _SC_GETGR_R_SIZE_MAX
		// and just return -1.  So just use the same
		// size that Linux returns.
		bufSize = 1024
	} else {
		bufSize = C.sysconf(C._SC_GETGR_R_SIZE_MAX)
		if bufSize <= 0 || bufSize > 1<<20 {
			return nil, fmt.Errorf("user: unreasonable _SC_GETGR_R_SIZE_MAX of %d", bufSize)
		}
	}
	buf := C.malloc(C.size_t(bufSize))
	defer C.free(buf)

	var rv C.int
	if lookupByName {
		nameC := C.CString(groupname)
		defer C.free(unsafe.Pointer(nameC))
		rv = C.getgrnam_r(nameC,
			&grp,
			(*C.char)(buf),
			C.size_t(bufSize),
			&result)
		if rv != 0 {
			return nil, fmt.Errorf("group: lookup group %s: %s", groupname, syscall.Errno(rv))
		}
		if result == nil {
			return nil, UnknownGroupError(groupname)
		}
	} else {
		// mygetgrgid_r is a wrapper around getgrgid_r to
		// to avoid using uid_t because C.uid_t(uid) for
		// unknown reasons doesn't work on linux.
		rv = C.mygetgrgid_r(C.int(gid),
			&grp,
			(*C.char)(buf),
			C.size_t(bufSize),
			&result)
		if rv != 0 {
			return nil, fmt.Errorf("grp: lookup groupid %d: %s", gid, syscall.Errno(rv))
		}
		if result == nil {
			return nil, UnknownGroupIdError(gid)
		}
	}

	members := make([]string, 0)
	for memptr := grp.gr_mem; *memptr != nil; memptr =
		(**C.char)(unsafe.Pointer(uintptr(unsafe.Pointer(memptr)) +
			unsafe.Sizeof(*grp.gr_mem))) {
		members = append(members, C.GoString(*memptr))
	}

	g := &Group{
		Gid:      strconv.Itoa(int(grp.gr_gid)),
		Name:     C.GoString(grp.gr_name),
		Password: C.GoString(grp.gr_passwd),
		Members:  members,
	}

	return g, nil
}
