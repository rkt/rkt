/* translate a prepared standard container in stage1 into systemd unit files */

package main

import (
	"fmt"
	"os"
	"strings"
	"path"
	"io"
	"syscall"

	"github.com/containers/standard/schema"
	"github.com/coreos/go-systemd/unit"
)


/* transform the provided app manifest into a systemd service unit */
func AppToSystemd(am *schema.AppManifest, befores []string) error {
	name := am.Name.String()
	opts := []*unit.UnitOption{
		&unit.UnitOption{"Unit",	"Description",		name},
		&unit.UnitOption{"Unit",	"DefaultDependencies",	"false"},
		&unit.UnitOption{"Service",	"Type",			am.Type.String()},
		&unit.UnitOption{"Service",	"Restart",		"no"},
		&unit.UnitOption{"Service",	"RootDirectory",	ServicePath(name)},
		&unit.UnitOption{"Service",	"ExecStart",		"\""+strings.Join(am.Exec, "\" \"")+"\""},
		&unit.UnitOption{"Service",	"User",			am.User},
		&unit.UnitOption{"Service",	"Group",		am.Group},
	}

	for ek, ev := range am.Environment {
		opts = append(opts, &unit.UnitOption{"Service", "Environment", "\""+ek+"="+ev+"\""})
	}

	for _, b := range befores {
		opts = append(opts, &unit.UnitOption{"Unit", "Before", ServicePath(b)})
	}

	file, err := os.OpenFile(ServiceFilePath(name), os.O_WRONLY|os.O_CREATE, 0640)
	if err != nil {
		return fmt.Errorf("Failed to create service file: %v", err)
	}
	defer file.Close()

	_, err = io.Copy(file, unit.Serialize(opts))
	if err != nil {
		return fmt.Errorf("Failed to write service file: %v", err)
	}

	err = syscall.Symlink(path.Join("..", ServicePath(name)), WantLinkPath(name))
	if err != nil {
		return fmt.Errorf("Failed to link service want: %v", err)
	}

	return nil
}


/* take an prepared scf execution group and output systemd service unit files */
func ContainerToSystemd(c *Container) error {
	for _, am := range c.Apps {
		name := am.Name.String()
		err := AppToSystemd(am, c.Manifest.Apps[name].Before)
		if err != nil {
			return fmt.Errorf("Failed to transform app \"%s\" into sysd service: %v", name, err)
		}
	}

	return nil
}


func main() {
	SetRootPath(".")

	c, err := LoadContainer()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load container: %v\n", err)
		os.Exit(1)
	}

	err = ContainerToSystemd(c)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to transform container into systemd units: %v\n", err)
		os.Exit(2)
	}
}
