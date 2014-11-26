package rkt

import (
	"path/filepath"
)

const (
	stage1Dir   = "/stage1"
	stage1Init  = stage1Dir + "/init"
	stage2Dir   = "/opt/stage2"
	servicesDir = stage1Dir + "/usr/lib/systemd/system"
	wantsDir    = servicesDir + "/default.target.wants"
)

// Stage1RootfsPath returns the directory in root containing the rootfs for stage1
func Stage1RootfsPath(root string) string {
	return filepath.Join(root, stage1Dir)
}

// Stage1InitPath returns the path to the file in root that is the stage1 init process
func Stage1InitPath(root string) string {
	return filepath.Join(root, stage1Init)
}

// ContainerManifestPath returns the path in root to the Container Runtime Manifest
func ContainerManifestPath(root string) string {
	return filepath.Join(root, "container")
}

// AppImagePath returns the path where an app image (i.e. RAF) is rooted (i.e.
// where its contents are extracted during stage0), based on the app image ID.
func AppImagePath(root string, id string) string {
	return filepath.Join(root, stage1Dir, stage2Dir, id)
}

// AppRootfsPath returns the path to an app's rootfs.
// id should be the app image ID.
func AppRootfsPath(root string, id string) string {
	return filepath.Join(AppImagePath(root, id), "rootfs")
}

// RelAppImagePath returns the path of an application image relative to the
// stage1 chroot
func RelAppImagePath(id string) string {
	return filepath.Join(stage2Dir, id)
}

// RelAppImagePath returns the path of an application's rootfs relative to the
// stage1 chroot
func RelAppRootfsPath(id string) string {
	return filepath.Join(RelAppImagePath(id), "rootfs")
}

// AppManifestPath returns the path to the app's manifest file inside the RAF.
// id should be the app image ID.
func AppManifestPath(root string, imageID string) string {
	return filepath.Join(AppImagePath(root, imageID), "manifest")
}

// WantsPath returns the systemd "wants" directory in root
func WantsPath(root string) string {
	return filepath.Join(root, wantsDir)
}

// ServicesPath returns the systemd "services" directory in root
func ServicesPath(root string) string {
	return filepath.Join(root, servicesDir)
}

// ServiceName returns a sanitized (escaped) systemd service name
// for the given appid
func ServiceName(appid string) string {
	return appid + ".service"
}

// ServiceFilePath returns the path to the systemd service file
// path for the given appid
func ServiceFilePath(root, appid string) string {
	return filepath.Join(root, servicesDir, ServiceName(appid))
}

// WantLinkPath returns the systemd "want" symlink path for the
// given appid
func WantLinkPath(root, appid string) string {
	return filepath.Join(root, wantsDir, ServiceName(appid))
}
