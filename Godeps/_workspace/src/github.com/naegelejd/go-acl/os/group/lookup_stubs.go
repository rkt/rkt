// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build !cgo,!windows,!plan9 android

package group

import (
	"fmt"
	"runtime"
)

func init() {
	implemented = false
}

func current() (*Group, error) {
	return nil, fmt.Errorf("group: Current not implemented on %s/%s", runtime.GOOS, runtime.GOARCH)
}

func lookup(groupname string) (*Group, error) {
	return nil, fmt.Errorf("group: Lookup not implemented on %s/%s", runtime.GOOS, runtime.GOARCH)
}

func lookupId(gid string) (*Group, error) {
	return nil, fmt.Errorf("group: LookupId not implemented on %s/%s", runtime.GOOS, runtime.GOARCH)
}
