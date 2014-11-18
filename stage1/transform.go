// translate a prepared standard container in stage1 into systemd unit files

package main

import (
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"github.com/containers/standard/schema"
	"github.com/containers/standard/schema/types"
	"github.com/coreos/go-systemd/unit"
)

// transform the provided app manifest into a systemd service unit
func (c *Container) appToSystemd(am *schema.AppManifest) error {
	typemap := map[string]string{"fork": "simple", "exit": "oneshot"}
	name := am.Name.String()
	opts := []*unit.UnitOption{
		&unit.UnitOption{"Unit", "Description", name},
		&unit.UnitOption{"Unit", "DefaultDependencies", "false"},
		&unit.UnitOption{"Service", "Type", typemap[am.StartedOn.String()]},
		&unit.UnitOption{"Service", "Restart", "no"},
		&unit.UnitOption{"Service", "RootDirectory", AppMountPath(name, true)},
		&unit.UnitOption{"Service", "ExecStart", "\"" + strings.Join(am.Exec, "\" \"") + "\""},
		&unit.UnitOption{"Service", "User", am.User},
		&unit.UnitOption{"Service", "Group", am.Group},
	}

	for ek, ev := range am.Environment {
		opts = append(opts, &unit.UnitOption{"Service", "Environment", "\"" + ek + "=" + ev + "\""})
	}

	file, err := os.OpenFile(ServiceFilePath(name, false), os.O_WRONLY|os.O_CREATE, 0640)
	if err != nil {
		return fmt.Errorf("failed to create service file: %v", err)
	}
	defer file.Close()

	_, err = io.Copy(file, unit.Serialize(opts))
	if err != nil {
		return fmt.Errorf("failed to write service file: %v", err)
	}

	if err = os.Symlink(path.Join("..", ServicePath(name)), WantLinkPath(name, false)); err != nil {
		return fmt.Errorf("failed to link service want: %v", err)
	}

	return nil
}

// take an prepared container execution group and output systemd service unit files
func (c *Container) ContainerToSystemd() error {
	if err := os.MkdirAll(path.Join(".", WantsDir), 0776); err != nil {
		return err
	}
	for _, am := range c.Apps {
		name := am.Name.String()
		if err := c.appToSystemd(am); err != nil {
			return fmt.Errorf("failed to transform app \"%s\" into sysd service: %v", name, err)
		}
	}

	return nil
}

// transform the provided app manifest into a subset of applicable systemd-nspawn arguments
func (c *Container) appToNspawnArgs(am *schema.AppManifest) ([]string, error) {
	args := []string{}
	name := am.Name.String()

	vols := make(map[types.ACLabel]types.Volume)
	for _, v := range c.Manifest.Volumes {
		for _, f := range v.Fulfills {
			vols[f] = v
		}
	}

	for key, mp := range am.MountPoints {
		vol, ok := vols[key]
		if !ok {
			return nil, fmt.Errorf("no volume for mountpoint %q in app %q", key, name)
		}
		opt := make([]string, 4)

		if mp.ReadOnly {
			opt[0] = "--bind-ro="
		} else {
			opt[0] = "--bind="
		}

		opt[1] = vol.Source
		opt[2] = ":"
		opt[3] = path.Join(AppRootfsPath(name, true), mp.Path)

		args = append(args, strings.Join(opt, ""))
	}

	return args, nil
}

// take an prepared container execution group and return a systemd-nspawn argument list
func (c *Container) ContainerToNspawnArgs() ([]string, error) {
	args := []string{
		"--boot",
		"--uuid=" + c.Manifest.UUID.String(),
		"--directory=" + Stage1RootfsPath(false),
	}

	for _, am := range c.Apps {
		aa, err := c.appToNspawnArgs(am)
		if err != nil {
			return nil, fmt.Errorf("failed to construct args for app %q: %v", am.Name, err)
		}
		args = append(args, aa...)
	}

	return args, nil
}
