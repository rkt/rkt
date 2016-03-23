// Copyright 2015 The rkt Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package common

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/appc/spec/schema"
	"github.com/appc/spec/schema/types"
	"github.com/coreos/rkt/common"
	"github.com/hashicorp/errwrap"
)

func MountSharedVolumes(root string, appName types.ACName, volumes []types.Volume, ra *schema.RuntimeApp) error {
	app := ra.App
	vols := make(map[types.ACName]types.Volume)
	for _, v := range volumes {
		vols[v.Name] = v
	}

	sharedVolPath := common.SharedVolumesPath(root)
	if err := os.MkdirAll(sharedVolPath, sharedVolPerm); err != nil {
		return errwrap.Wrap(errors.New("could not create shared volumes directory"), err)
	}
	if err := os.Chmod(sharedVolPath, sharedVolPerm); err != nil {
		return errwrap.Wrap(fmt.Errorf("could not change permissions of %q", sharedVolPath), err)
	}

	mounts := generateMounts(ra, vols)
	for _, m := range mounts {
		vol := vols[m.Volume]

		if vol.Kind == "empty" {
			p := filepath.Join(sharedVolPath, vol.Name.String())
			if err := os.MkdirAll(p, sharedVolPerm); err != nil {
				return errwrap.Wrap(fmt.Errorf("could not create shared volume %q", vol.Name), err)
			}
			if err := os.Chown(p, *vol.UID, *vol.GID); err != nil {
				return errwrap.Wrap(fmt.Errorf("could not change owner of %q", p), err)
			}
			mod, err := strconv.ParseUint(*vol.Mode, 8, 32)
			if err != nil {
				return errwrap.Wrap(fmt.Errorf("invalid mode %q for volume %q", *vol.Mode, vol.Name), err)
			}
			if err := os.Chmod(p, os.FileMode(mod)); err != nil {
				return errwrap.Wrap(fmt.Errorf("could not change permissions of %q", p), err)
			}
		}

		readOnly := IsMountReadOnly(vol, app.MountPoints)
		var source string
		switch vol.Kind {
		case "host":
			source = vol.Source
		case "empty":
			source = filepath.Join(common.SharedVolumesPath(root), vol.Name.String())
		default:
			return fmt.Errorf(`invalid volume kind %q. Must be one of "host" or "empty"`, vol.Kind)
		}
		appRootfs := common.AppRootfsPath(".", appName)
		mntPath, err := evaluateAppMountPath(appRootfs, m.Path)
		if err != nil {
			return errwrap.Wrap(fmt.Errorf("could not evaluate path %v", m.Path), err)
		}

		destination := filepath.Join(appRootfs, mntPath)

		if err := doBindMount(source, destination, readOnly); err != nil {
			return errwrap.Wrap(fmt.Errorf("could not bind mount path %v (s: %v, d: %v)", m.Path, source, destination), err)
		}
	}
	return nil
}

func doBindMount(source, destination string, readOnly bool) error {
	if err := syscall.Mount(source, destination, "bind", syscall.MS_BIND, ""); err != nil {
		return err
	}
	if readOnly {
		return syscall.Mount(source, destination, "bind", syscall.MS_REMOUNT|syscall.MS_RDONLY|syscall.MS_BIND, "")
	}
	return nil
}
