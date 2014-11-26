package types

import (
	"encoding/json"
	"fmt"
	"strings"
)

const (
	valchars = `abcdefghijklmnopqrstuvwxyz0123456789.-/`
)

// ACName (an App-Container Name) is a format used by keys in different
// formats of the App Container Standard. An ACName is restricted to
// characters accepted by the DNS RFC[1] and "/". ACNames are
// case-insensitive for comparison purposes, but case-preserving.
//
// [1] http://tools.ietf.org/html/rfc1123#page-13
type ACName string

func (l ACName) String() string {
	return string(l)
}

// Equals checks whether a given ACName is equal to this one.
func (l ACName) Equals(o ACName) bool {
	return strings.ToLower(string(l)) == strings.ToLower(string(o))
}

// NewACName generates a new ACName from a string. If the given string is
// not a valid ACName, nil and an error are returned.
func NewACName(s string) (*ACName, error) {
	for _, c := range s {
		if !strings.ContainsRune(valchars, c) {
			msg := fmt.Sprintf("invalid char in ACName: %c", c)
			return nil, ACNameError(msg)
		}
	}
	return (*ACName)(&s), nil
}

// UnmarshalJSON implements the json.Unmarshaler interface
func (l *ACName) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	nl, err := NewACName(s)
	if err != nil {
		return err
	}
	*l = *nl
	return nil
}

// MarshalJSON implements the json.Marshaler interface
func (l *ACName) MarshalJSON() ([]byte, error) {
	return json.Marshal(l.String())
}
