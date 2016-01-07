// Package group allows group lookups by name or id.
package group

var implemented = true // set to false by lookup_stubs.go's init

// Group represents a group
//
// On posix systems Gid contains a decimal number
// representing gid. On windows Gid contain security
// identifier (SID) in a string format. On Plan 9,
// Gid, and Name will be the contents of /dev/group.
type Group struct {
	Gid      string
	Name     string
	Password string
	Members  []string
}

// UnknownGroupIdError is returned by LookupId when
// a group cannot be found.
type UnknownGroupIdError int

func (e UnknownGroupIdError) Error() string {
	return "unknown group id"
}

// UnknownGroupError is returned by Lookup when
// a group cannot be found
type UnknownGroupError string

func (e UnknownGroupError) Error() string {
	return "unknown group name"
}
