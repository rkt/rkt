// Copyright 2016 The rkt Authors
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

//+build linux

package stage0

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/coreos/rkt/common"
	"github.com/coreos/rkt/common/apps"
	pkgPod "github.com/coreos/rkt/pkg/pod"
	"github.com/coreos/rkt/pkg/user"

	"github.com/appc/spec/schema"
	"github.com/appc/spec/schema/types"
	"github.com/hashicorp/errwrap"
)

type StartConfig struct {
	*CommonConfig
	PodPath     string
	UsesOverlay bool
	AppName     *types.ACName
	PodPID      int
}

type StopConfig struct {
	*CommonConfig
	PodPath string
	AppName *types.ACName
	PodPID  int
}

type AddConfig struct {
	*CommonConfig
	Image                types.Hash
	Apps                 *apps.Apps
	RktGid               int
	UsesOverlay          bool
	PodPath              string
	PodPID               int
	InsecureCapabilities bool
	InsecurePaths        bool
	InsecureSeccomp      bool
}

type RmConfig struct {
	*CommonConfig
	PodPath     string
	UsesOverlay bool
	AppName     *types.ACName
	PodPID      int
}

func AddApp(cfg AddConfig) error {
	// there should be only one app in the config
	app := cfg.Apps.Last()
	if app == nil {
		return errors.New("no image specified")
	}

	am, err := cfg.Store.GetImageManifest(cfg.Image.String())
	if err != nil {
		return err
	}

	var appName *types.ACName
	if app.Name != "" {
		appName, err = types.NewACName(app.Name)
		if err != nil {
			return err
		}
	} else {
		appName, err = imageNameToAppName(am.Name)
		if err != nil {
			return err
		}
	}

	pod, err := pkgPod.PodFromUUIDString(cfg.DataDir, cfg.UUID.String())
	if err != nil {
		return errwrap.Wrap(errors.New("error loading pod"), err)
	}
	defer pod.Close()

	debug("locking pod manifest")
	if err := pod.ExclusiveLockManifest(); err != nil {
		return errwrap.Wrap(errors.New("failed to lock pod manifest"), err)
	}
	defer pod.UnlockManifest()

	pm, err := pod.SandboxManifest()
	if err != nil {
		return errwrap.Wrap(errors.New("cannot add application"), err)
	}

	if pm.Apps.Get(*appName) != nil {
		return fmt.Errorf("error: multiple apps with name %s", *appName)
	}

	if am.App == nil && app.Exec == "" {
		return fmt.Errorf("error: image %s has no app section and --exec argument is not provided", cfg.Image)
	}

	appInfoDir := common.AppInfoPath(cfg.PodPath, *appName)
	if err := os.MkdirAll(appInfoDir, common.DefaultRegularDirPerm); err != nil {
		return errwrap.Wrap(errors.New("error creating apps info directory"), err)
	}

	pcfg := PrepareConfig{
		CommonConfig: cfg.CommonConfig,
		PrivateUsers: user.NewBlankUidRange(),
	}

	if cfg.UsesOverlay {
		privateUsers, err := preparedWithPrivateUsers(cfg.PodPath)
		if err != nil {
			log.FatalE("error reading user namespace information", err)
		}

		if err := pcfg.PrivateUsers.Deserialize([]byte(privateUsers)); err != nil {
			return err
		}
	}

	treeStoreID, err := prepareAppImage(pcfg, *appName, cfg.Image, cfg.PodPath, cfg.UsesOverlay)
	if err != nil {
		return errwrap.Wrap(fmt.Errorf("error preparing image %s", cfg.Image), err)
	}

	rcfg := RunConfig{
		CommonConfig: cfg.CommonConfig,
		UseOverlay:   cfg.UsesOverlay,
		RktGid:       cfg.RktGid,
	}

	if err := setupAppImage(rcfg, *appName, cfg.Image, cfg.PodPath, cfg.UsesOverlay); err != nil {
		return fmt.Errorf("error setting up app image: %v", err)
	}

	if cfg.UsesOverlay {
		imgDir := filepath.Join(cfg.PodPath, "overlay", treeStoreID)
		if err := os.Chown(imgDir, -1, cfg.RktGid); err != nil {
			return err
		}
	}

	ra := schema.RuntimeApp{
		Name: *appName,
		App:  am.App,
		Image: schema.RuntimeImage{
			Name:   &am.Name,
			ID:     cfg.Image,
			Labels: am.Labels,
		},
		Mounts:         MergeMounts(cfg.Apps.Mounts, app.Mounts),
		ReadOnlyRootFS: app.ReadOnlyRootFS,
	}

	if app.Exec != "" {
		// Create a minimal App section if not present
		if am.App == nil {
			ra.App = &types.App{
				User:  strconv.Itoa(os.Getuid()),
				Group: strconv.Itoa(os.Getgid()),
			}
		}
		ra.App.Exec = []string{app.Exec}
	}

	if app.Args != nil {
		ra.App.Exec = append(ra.App.Exec, app.Args...)
	}

	if app.WorkingDir != "" {
		ra.App.WorkingDirectory = app.WorkingDir
	}

	if err := prepareIsolators(app, ra.App); err != nil {
		return err
	}

	if app.User != "" {
		ra.App.User = app.User
	}

	if app.Group != "" {
		ra.App.Group = app.Group
	}

	if app.SupplementaryGIDs != nil {
		ra.App.SupplementaryGIDs = app.SupplementaryGIDs
	}

	if app.UserAnnotations != nil {
		ra.App.UserAnnotations = app.UserAnnotations
	}

	if app.UserLabels != nil {
		ra.App.UserLabels = app.UserLabels
	}

	if app.Environments != nil {
		envs := make([]string, 0, len(app.Environments))
		for name, value := range app.Environments {
			envs = append(envs, fmt.Sprintf("%s=%s", name, value))
		}
		// Let the app level environment override the environment variables.
		mergeEnvs(&ra.App.Environment, envs, true)
	}

	env := ra.App.Environment

	env.Set("AC_APP_NAME", appName.String())
	envFilePath := filepath.Join(common.Stage1RootfsPath(cfg.PodPath), "rkt", "env", appName.String())

	if err := common.WriteEnvFile(env, pcfg.PrivateUsers, envFilePath); err != nil {
		return err
	}

	debug("adding app to sandbox")
	pm.Apps = append(pm.Apps, ra)
	if err := pod.UpdateManifest(pm, cfg.PodPath); err != nil {
		return err
	}

	args := []string{
		fmt.Sprintf("--debug=%t", cfg.Debug),
		fmt.Sprintf("--uuid=%s", cfg.UUID),
		fmt.Sprintf("--app=%s", appName),
	}

	if _, err := os.Create(common.AppCreatedPath(pod.Path(), appName.String())); err != nil {
		return err
	}

	ce := CrossingEntrypoint{
		PodPath:        cfg.PodPath,
		PodPID:         cfg.PodPID,
		AppName:        appName.String(),
		EntrypointName: appAddEntrypoint,
		EntrypointArgs: args,
		Interactive:    false,
	}
	if err := ce.Run(); err != nil {
		return err
	}

	return nil
}

func RmApp(cfg RmConfig) error {
	pod, err := pkgPod.PodFromUUIDString(cfg.DataDir, cfg.UUID.String())
	if err != nil {
		return errwrap.Wrap(errors.New("error loading pod"), err)
	}
	defer pod.Close()

	debug("locking sandbox manifest")
	if err := pod.ExclusiveLockManifest(); err != nil {
		return errwrap.Wrap(errors.New("failed to lock sandbox manifest"), err)
	}
	defer pod.UnlockManifest()

	pm, err := pod.SandboxManifest()
	if err != nil {
		return errwrap.Wrap(errors.New("cannot remove application, sandbox validation failed"), err)
	}

	app := pm.Apps.Get(*cfg.AppName)
	if app == nil {
		return fmt.Errorf("error: nonexistent app %q", *cfg.AppName)
	}

	if cfg.PodPID > 0 {
		// Call app-stop and app-rm entrypoint only if the pod is still running.
		// Otherwise, there's not much we can do about it except unmounting/removing
		// the file system.
		args := []string{
			fmt.Sprintf("--debug=%t", cfg.Debug),
			fmt.Sprintf("--app=%s", cfg.AppName),
		}

		ce := CrossingEntrypoint{
			PodPath:        cfg.PodPath,
			PodPID:         cfg.PodPID,
			AppName:        cfg.AppName.String(),
			EntrypointName: appStopEntrypoint,
			EntrypointArgs: args,
			Interactive:    false,
		}
		if err := ce.Run(); err != nil {
			status, err := common.GetExitStatus(err)
			// ignore nonexistent units failing to stop. Exit status 5
			// comes from systemctl and means the unit doesn't exist
			if err != nil {
				return err
			} else if status != 5 {
				return fmt.Errorf("exit status %d", status)
			}
		}

		ce.EntrypointName = appRmEntrypoint
		if err := ce.Run(); err != nil {
			return err
		}
	}

	if cfg.UsesOverlay {
		treeStoreID, err := ioutil.ReadFile(common.AppTreeStoreIDPath(cfg.PodPath, *cfg.AppName))
		if err != nil {
			return err
		}

		appRootfs := common.AppRootfsPath(cfg.PodPath, *cfg.AppName)
		if err := syscall.Unmount(appRootfs, 0); err != nil {
			return err
		}

		ts := filepath.Join(cfg.PodPath, "overlay", string(treeStoreID))
		if err := os.RemoveAll(ts); err != nil {
			return errwrap.Wrap(errors.New("error removing app info directory"), err)
		}
	}

	appInfoDir := common.AppInfoPath(cfg.PodPath, *cfg.AppName)
	if err := os.RemoveAll(appInfoDir); err != nil {
		return errwrap.Wrap(errors.New("error removing app info directory"), err)
	}

	if err := os.RemoveAll(common.AppPath(cfg.PodPath, *cfg.AppName)); err != nil {
		return err
	}

	appStatusPath := filepath.Join(common.Stage1RootfsPath(cfg.PodPath), "rkt", "status", cfg.AppName.String())
	if err := os.Remove(appStatusPath); err != nil && !os.IsNotExist(err) {
		return err
	}

	envPath := filepath.Join(common.Stage1RootfsPath(cfg.PodPath), "rkt", "env", cfg.AppName.String())
	if err := os.Remove(envPath); err != nil && !os.IsNotExist(err) {
		return err
	}

	for i, app := range pm.Apps {
		if app.Name == *cfg.AppName {
			pm.Apps = append(pm.Apps[:i], pm.Apps[i+1:]...)
			break
		}
	}

	return pod.UpdateManifest(pm, cfg.PodPath)
}

func StartApp(cfg StartConfig) error {
	pod, err := pkgPod.PodFromUUIDString(cfg.DataDir, cfg.UUID.String())
	if err != nil {
		return errwrap.Wrap(errors.New("error loading pod"), err)
	}
	defer pod.Close()

	pm, err := pod.SandboxManifest()
	if err != nil {
		return errwrap.Wrap(errors.New("cannot start application"), err)
	}

	app := pm.Apps.Get(*cfg.AppName)
	if app == nil {
		return fmt.Errorf("error: nonexistent app %q", *cfg.AppName)
	}

	args := []string{
		fmt.Sprintf("--debug=%t", cfg.Debug),
		fmt.Sprintf("--app=%s", cfg.AppName),
	}

	if _, err := os.Create(common.AppStartedPath(cfg.PodPath, cfg.AppName.String())); err != nil {
		log.FatalE(fmt.Sprintf("error creating %s-started file", cfg.AppName.String()), err)
	}

	ce := CrossingEntrypoint{
		PodPath:        cfg.PodPath,
		PodPID:         cfg.PodPID,
		AppName:        cfg.AppName.String(),
		EntrypointName: appStartEntrypoint,
		EntrypointArgs: args,
		Interactive:    false,
	}
	if err := ce.Run(); err != nil {
		return err
	}

	return nil
}

func StopApp(cfg StopConfig) error {
	pod, err := pkgPod.PodFromUUIDString(cfg.DataDir, cfg.UUID.String())
	if err != nil {
		return errwrap.Wrap(errors.New("error loading pod manifest"), err)
	}
	defer pod.Close()

	pm, err := pod.SandboxManifest()
	if err != nil {
		return errwrap.Wrap(errors.New("cannot stop application"), err)
	}

	app := pm.Apps.Get(*cfg.AppName)
	if app == nil {
		return fmt.Errorf("error: nonexistent app %q", *cfg.AppName)
	}

	args := []string{
		fmt.Sprintf("--debug=%t", cfg.Debug),
		fmt.Sprintf("--app=%s", cfg.AppName),
	}

	ce := CrossingEntrypoint{
		PodPath:        cfg.PodPath,
		PodPID:         cfg.PodPID,
		AppName:        cfg.AppName.String(),
		EntrypointName: appStopEntrypoint,
		EntrypointArgs: args,
		Interactive:    false,
	}
	if err := ce.Run(); err != nil {
		status, err := common.GetExitStatus(err)
		// exit status 5 comes from systemctl and means the unit doesn't exist
		if status == 5 {
			return fmt.Errorf("app %q is not running", app.Name)
		}

		return err
	}

	return nil
}
