/* translate an scf execution group into systemd units */

package main

import (
	"fmt"
	"os"
	"scf"
)


/* take an prepared scf execution group and output systemd unit files */
func eg2sdu(eg *scf.ExecGroup, svc_dir string) error {
	return nil
}

/* assume our CWD is the stage1 root */
func main() {
	eg, err := scf.LoadExecGroup(".")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load exec group")
		os.Exit(1)
	}

	eg2sdu(eg, "/usr/systemd/system")
}
