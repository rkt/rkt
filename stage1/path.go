package main

import (
	"path/filepath"

	"github.com/coreos/go-systemd/dbus"
)

const (
	ContainerFile = "/container"
	Stage1Dir     = "/stage1"
	ServicesDir   = Stage1Dir + "/run/systemd/system"
	WantsDir      = ServicesDir + "/default.target.wants"
	ServiceSuffix = ".service"
	Stage2Dir     = Stage1Dir + "/opt/stage2"
	AppsSubdir    = "apps"
	RootfsDir     = "rootfs"
)

var (
	rootPath = "."
)

// set where stage1 is rooted
// stage1 will execute with the cwd set to the root, but arbitrary prefixes are handy for test/dev.
func SetRootPath(root string) {
	rootPath = root
}

func rootedPath(path string, chroot bool) string {
	if chroot == false {
		return filepath.Join(rootPath, path)
	}
	return path
}

// returns the path to the stage1 rootfs path
func Stage1RootfsPath(chroot bool) string {
	return rootedPath(Stage1Dir, chroot)
}

// returns the container manifest path
func ContainerManifestPath(chroot bool) string {
	return rootedPath(ContainerFile, chroot)
}

// returns the path where the named app is rooted i.e. where its contents are extracted within stage1
// Mount is used instead of Root to avoid confusion with the app's rootfs directory
func AppMountPath(name string, chroot bool) string {
	esc := dbus.PathBusEscape(name)
	return rootedPath(filepath.Join(Stage2Dir, esc), chroot)
}

// returns the path to the named app's rootfs
func AppRootfsPath(name string, chroot bool) string {
	return filepath.Join(AppMountPath(name, chroot), RootfsDir)
}

// returns the path to the app manifest file within stage1
func AppManifestPath(name string, chroot bool) string {
	return filepath.Join(AppMountPath(name, chroot), AppsSubdir, name)
}

// returns the systemd service path for the named app
// XXX this doesn't quite mesh in here
func ServicePath(name string) string {
	return dbus.PathBusEscape(name) + ServiceSuffix
}

// returns the systemd want symlink path for the named app
func WantLinkPath(name string, chroot bool) string {
	return rootedPath(filepath.Join(WantsDir, ServicePath(name)), chroot)
}

// returns the path to the systemd service file path for the named app
func ServiceFilePath(name string, chroot bool) string {
	return rootedPath(filepath.Join(ServicesDir, ServicePath(name)), chroot)
}
