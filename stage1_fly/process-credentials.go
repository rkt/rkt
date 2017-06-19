package stage1_fly

import (
	"fmt"
	"runtime"
	"syscall"

	"github.com/appc/spec/schema"
	"github.com/hashicorp/errwrap"
	rktlog "github.com/rkt/rkt/pkg/log"
	"github.com/rkt/rkt/pkg/user"
)

type ProcessCredentials struct {
	uid               int
	gid               int
	supplementaryGIDs []int
}

func LookupProcessCredentials(ra *schema.RuntimeApp, rfs string) (*ProcessCredentials, error) {
	var c ProcessCredentials

	uidResolver, err := user.NumericIDs(ra.App.User)
	if err != nil {
		uidResolver, err = user.IDsFromStat(rfs, ra.App.User, nil)
	}

	if err != nil { // give up
		return nil, errwrap.Wrap(fmt.Errorf("invalid user %q", ra.App.User), err)
	}

	if c.uid, _, err = uidResolver.IDs(); err != nil {
		return nil, errwrap.Wrap(fmt.Errorf("failed to configure user %q", ra.App.User), err)
	}

	gidResolver, err := user.NumericIDs(ra.App.Group)
	if err != nil {
		gidResolver, err = user.IDsFromStat(rfs, ra.App.Group, nil)
	}

	if err != nil { // give up
		return nil, errwrap.Wrap(fmt.Errorf("invalid group %q", ra.App.Group), err)
	}

	if _, c.gid, err = gidResolver.IDs(); err != nil {
		return nil, errwrap.Wrap(fmt.Errorf("failed to configure group %q", ra.App.Group), err)
	}

	// supplementary groups - ensure primary group is included
	n := len(ra.App.SupplementaryGIDs)
	c.supplementaryGIDs = make([]int, n+1, n+1)
	c.supplementaryGIDs[0] = c.gid
	copy(c.supplementaryGIDs[1:], ra.App.SupplementaryGIDs)

	return &c, nil
}

func SetProcessCredentials(c *ProcessCredentials, diag *rktlog.Logger) error {
	// lock the current goroutine to its current OS thread.
	// This will force the subsequent syscalls to be executed in the same OS thread as Setresuid, and Setresgid,
	// see https://github.com/golang/go/issues/1435#issuecomment-66054163.
	runtime.LockOSThread()

	diag.Printf("setting credentials: uid=%d, gid=%d, suppGids=%v", c.uid, c.gid, c.supplementaryGIDs)
	if err := syscall.Setresgid(c.gid, c.gid, c.gid); err != nil {
		return err
	}
	if err := syscall.Setgroups(c.supplementaryGIDs); err != nil {
		return err
	}
	if err := syscall.Setresuid(c.uid, c.uid, c.uid); err != nil {
		return err
	}

	return nil
}
