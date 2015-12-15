package acl

// #ifdef __APPLE__
//  #include <sys/types.h>
// #endif
// #include <sys/acl.h>
// #cgo linux LDFLAGS: -lacl
import "C"
import "fmt"

const (
	PermExecute Perm = C.ACL_EXECUTE
)

// Perm represents a permission.
type Perm int

// Permset is a collection of permissions.
type Permset struct {
	p C.acl_permset_t
}

// AddPerm adds a new permission to a Permset.
func (pset *Permset) AddPerm(perm Perm) error {
	rv, _ := C.acl_add_perm(pset.p, C.acl_perm_t(perm))
	if rv < 0 {
		return fmt.Errorf("unable to add perm to permset")
	}
	return nil
}

// ClearPerms removes all permissions from a Permset.
func (pset *Permset) ClearPerms() error {
	rv, _ := C.acl_clear_perms(pset.p)
	if rv < 0 {
		return fmt.Errorf("unable to clear perms")
	}
	return nil
}

// DeletePerm removes a single permission from a Permset.
func (pset *Permset) DeletePerm(perm Perm) error {
	p := C.acl_perm_t(perm)
	rv, _ := C.acl_delete_perm(pset.p, p)
	if rv < 0 {
		return fmt.Errorf("unable to delete perm")
	}
	return nil
}
