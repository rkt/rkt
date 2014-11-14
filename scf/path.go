package main

import (
	"path/filepath"

	"github.com/coreos/go-systemd/dbus"
)

const (
	ContainerDir  = "/container"
	Stage1Dir     = "/stage1"
	ServicesDir   = Stage1Dir + "/run/systemd/system"
	WantsDir      = ServicesDir + "/default.target.wants"
	ServiceSuffix = ".service"
	Stage2Dir     = Stage1Dir + "/opt/stage2"
	AppsSubdir    = "apps"
)

var (
	rootPath = "."
)

// set where stage1 is rooted
// stage1 will execute with the cwd set to the root, but arbitrary prefixes are handy for test/dev.
func SetRootPath(root string) {
	rootPath = root
}

// returns the container manifest path
func ContainerManifestPath() string {
	return filepath.Join(rootPath, ContainerDir)
}

// returns the path where the named app is rooted i.e. where its contents are extracted within stage1
// used Mount instead of Root to avoid confusion with the apps rootfs
func AppMountPath(name string) string {
	esc := dbus.PathBusEscape(name)
	return filepath.Join(rootPath, Stage2Dir, esc)
}

// returns the path to the app manifest file within stage1
func AppManifestPath(name string) string {
	return filepath.Join(AppMountPath(name), AppsSubdir, name)
}

// returns the systemd service path for the named app
func ServicePath(name string) string {
	return dbus.PathBusEscape(name) + ServiceSuffix
}

// returns the systemd want symlink path for the named app
func WantLinkPath(name string) string {
	return filepath.Join(rootPath, WantsDir, ServicePath(name))
}

// returns the path to the systemd service file path for the named app
func ServiceFilePath(name string) string {
	return filepath.Join(rootPath, ServicesDir, ServicePath(name))
}
