package acl

// #ifdef __APPLE__
//  #include <sys/types.h>
// #endif
// #include <sys/acl.h>
// #cgo linux LDFLAGS: -lacl
import "C"
import (
	"fmt"
	"unsafe"
)

const (
	TagUndefined Tag = C.ACL_UNDEFINED_TAG
)

// Entry is an entry in an ACL.
type Entry struct {
	e C.acl_entry_t
}

// SetPermset sets the permissions for an ACL Entry.
func (entry *Entry) SetPermset(pset *Permset) error {
	rv, _ := C.acl_set_permset(entry.e, pset.p)
	if rv < 0 {
		return fmt.Errorf("unable to set permset on entry")
	}
	return nil
}

// Copy copies an Entry.
func (entry *Entry) Copy() (*Entry, error) {
	var cdst C.acl_entry_t
	rv, _ := C.acl_copy_entry(cdst, entry.e)
	if rv < 0 {
		return nil, fmt.Errorf("unable to copy entry")
	}
	return &Entry{cdst}, nil
}

// SetQualifier sets the Uid or Gid the entry applies to.
func (entry *Entry) SetQualifier(id int) error {
	rv, _ := C.acl_set_qualifier(entry.e, unsafe.Pointer(&id))
	if rv < 0 {
		return fmt.Errorf("unable to set qualifier")
	}
	return nil
}

// GetQualifier returns the Uid or Gid the entry applies to.
func (entry *Entry) GetQualifier() (int, error) {
	var id int
	rv, _ := C.acl_set_qualifier(entry.e, unsafe.Pointer(&id))
	if rv < 0 {
		return -1, fmt.Errorf("unable to get qualifier")
	}
	return id, nil
}

// GetPermset returns the permission for an Entry.
func (entry *Entry) GetPermset() (*Permset, error) {
	var ps C.acl_permset_t
	rv, _ := C.acl_get_permset(entry.e, &ps)
	if rv < 0 {
		return nil, fmt.Errorf("unable to get permset")
	}
	return &Permset{ps}, nil
}

// GetTag returns the Tag for an Entry.
func (entry *Entry) GetTag() (Tag, error) {
	var t C.acl_tag_t
	rv, _ := C.acl_get_tag_type(entry.e, &t)
	if rv < 0 {
		return TagUndefined, fmt.Errorf("unable to get tag")
	}
	return Tag(t), nil
}

// SetTag sets the Tag for an Entry.
func (entry *Entry) SetTag(t Tag) error {
	rv, _ := C.acl_set_tag_type(entry.e, C.acl_tag_t(t))
	if rv < 0 {
		return fmt.Errorf("unable to set tag")
	}
	return nil
}
