// Derived from os/user/lookup.go in the Go source code
// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
// - Joseph Naegele 2015

package group

// Current returns the current user's group.
func Current() (*Group, error) {
	return current()
}

// Lookup looks up a group by name. If the group cannot be found,
// the returned error is of type UnknownGroupError.
func Lookup(groupname string) (*Group, error) {
	return lookup(groupname)
}

// LookupId looks up a group by groupid. If the group cannot be found, the
// returned error is of type UnknownGroupIdError.
func LookupId(gid string) (*Group, error) {
	return lookupId(gid)
}
