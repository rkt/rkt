package sys

import (
	"os"

	"github.com/coreos/rkt/Godeps/_workspace/src/github.com/syndtr/gocapability/capability"
)

// HasChrootCapability checks if the current process has the CAP_SYS_CHROOT
// capability
func HasChrootCapability() bool {
	// Checking the capabilities should be enough, but in case there're
	// problem retrieving them, fallback checking for the effective uid
	// (hoping it hasn't dropped its CAP_SYS_CHROOT).
	caps, err := capability.NewPid(0)
	if err == nil {
		return caps.Get(capability.EFFECTIVE, capability.CAP_SYS_CHROOT)
	} else {
		return os.Geteuid() == 0
	}
}
