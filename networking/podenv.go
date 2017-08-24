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

package networking

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"syscall"

	"github.com/appc/spec/schema/types"
	"github.com/containernetworking/cni/pkg/ns"
	"github.com/hashicorp/errwrap"
	"github.com/rkt/rkt/common"
	"github.com/rkt/rkt/networking/netinfo"
	"golang.org/x/sys/unix"
)

const (
	// Suffix to LocalConfigDir path, where users place their net configs
	UserNetPathSuffix = "net.d"

	// Default net path relative to stage1 root
	BuiltinNetPath = "etc/rkt/" + UserNetPathSuffix
)

// "base" struct that's populated from the beginning
// describing the environment in which the pod
// is running in
type podEnv struct {
	podRoot      string
	podID        types.UUID
	netsLoadList common.NetList
	localConfig  string
	podNS        ns.NetNS
}

type activeNet struct {
	confBytes []byte
	conf      *NetConf
	runtime   *netinfo.NetInfo
}

type byFilename []activeNet

func (s byFilename) Len() int      { return len(s) }
func (s byFilename) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s byFilename) Less(i, j int) bool {
	fiName := filepath.Base(s[i].runtime.ConfPath)
	fjName := filepath.Base(s[j].runtime.ConfPath)
	return strings.Compare(fiName, fjName) < 0
}

// Loads nets specified by user, both from a configurable user location and builtin from stage1. User supplied network
// configs override what is built into stage1.
// The order in which networks are applied to pods will be defined by their filenames.
func (e *podEnv) loadNets() ([]activeNet, error) {
	if e.netsLoadList.None() {
		stderr.Printf("networking namespace with loopback only")
		return nil, nil
	}

	nets, err := e.newNetLoader().loadNets(e.netsLoadList)
	if err != nil {
		return nil, err
	}
	netSlice := make([]activeNet, 0, len(nets))
	for _, net := range nets {
		netSlice = append(netSlice, net)
	}
	sort.Sort(byFilename(netSlice))

	missing := missingNets(e.netsLoadList, netSlice)
	if len(missing) > 0 {
		return nil, fmt.Errorf("networks not found: %v", strings.Join(missing, ", "))
	}

	// Add the runtime args to the network instances.
	// We don't do this earlier because we also load networks in other contexts
	for _, n := range nets {
		n.runtime.Args = e.netsLoadList.SpecificArgs(n.conf.Name)
	}
	return netSlice, nil
}

// Ensure the netns directory is mounted before adding new netns like `ip netns add <netns>` command does.
// See https://github.com/kubernetes/kubernetes/issues/48427
// Make it possible for network namespace mounts to propagate between mount namespaces.
// This makes it likely that an unmounting a network namespace file in one namespace will unmount the network namespace.
// file in all namespaces allowing the network namespace to be freed sooner.
func (e *podEnv) mountNetnsDirectory() error {
	err := os.MkdirAll(mountNetnsDirectory, 0755)
	if err != nil {
		return err
	}

	err = syscall.Mount("", mountNetnsDirectory, "none", syscall.MS_SHARED|syscall.MS_REC, "")
	if err != nil {
		// Fail unless we need to make the mount point
		if err != syscall.EINVAL {
			return fmt.Errorf("mount --make-rshared %s failed: %q", mountNetnsDirectory, err)
		}

		// Upgrade mountTarget to a mount point
		err = syscall.Mount(mountNetnsDirectory, mountNetnsDirectory, "none", syscall.MS_BIND|syscall.MS_REC, "")
		if err != nil {
			return fmt.Errorf("mount --rbind %s %s failed: %q", mountNetnsDirectory, mountNetnsDirectory, err)
		}

		// Remount after the Upgrade
		err = syscall.Mount("", mountNetnsDirectory, "none", syscall.MS_SHARED|syscall.MS_REC, "")
		if err != nil {
			return fmt.Errorf("mount --make-rshared %s failed: %q", mountNetnsDirectory, err)
		}
	}
	return nil
}

// podNSCreate creates the network namespace and saves a reference to its path.
// NewNS will bind-mount the namespace in /run/netns, so we write that filename
// to disk.
func (e *podEnv) podNSCreate() error {
	podNS, err := ns.NewNS()
	if err != nil {
		return err
	}
	e.podNS = podNS

	if err := e.podNSPathSave(); err != nil {
		return err
	}
	return nil
}

func (e *podEnv) podNSFilePath() string {
	return filepath.Join(e.podRoot, "netns")
}

func (e *podEnv) podNSPathLoad() (string, error) {
	podNSPath, err := ioutil.ReadFile(e.podNSFilePath())
	if err != nil {
		return "", err
	}

	return string(podNSPath), nil
}

func podNSerrorOK(podNSPath string, err error) bool {
	switch err.(type) {
	case ns.NSPathNotExistErr:
		return true
	case ns.NSPathNotNSErr:
		return true

	default:
		if os.IsNotExist(err) {
			return true
		}
		return false
	}
}

func (e *podEnv) podNSLoad() error {
	podNSPath, err := e.podNSPathLoad()
	if err != nil && !podNSerrorOK(podNSPath, err) {
		return err
	} else {
		podNS, err := ns.GetNS(podNSPath)
		if err != nil && !podNSerrorOK(podNSPath, err) {
			return err
		}
		e.podNS = podNS
		return nil
	}
}

func (e *podEnv) podNSPathSave() error {
	podNSFile, err := os.OpenFile(e.podNSFilePath(), os.O_WRONLY|os.O_CREATE, 0)
	if err != nil {
		return err
	}
	defer podNSFile.Close()

	if _, err = io.WriteString(podNSFile, e.podNS.Path()); err != nil {
		return err
	}

	return nil
}

func (e *podEnv) podNSDestroy() error {
	if e.podNS == nil {
		return nil
	}

	// Close the namespace handle
	// If this handle also *created* the namespace, it will delete it for us.
	_ = e.podNS.Close()

	// We still need to try and delete the namespace ourselves - no way to know
	// if podNS.Close() did it for us.
	// Unmount the ns bind-mount, and delete the mountpoint if successful

	if err := syscall.Unmount(e.podNS.Path(), unix.MNT_DETACH); err != nil {
		// if already unmounted, umount(2) returns EINVAL - continue
		if !os.IsNotExist(err) && err != syscall.EINVAL {
			return errwrap.Wrap(fmt.Errorf("error unmounting netns %q", e.podNS.Path()), err)
		}
	}
	if err := os.RemoveAll(e.podNS.Path()); err != nil {
		if !os.IsNotExist(err) {
			return errwrap.Wrap(fmt.Errorf("failed to remove netns %s", e.podNS.Path()), err)
		}
	}
	return nil
}

func (e *podEnv) netDir() string {
	return filepath.Join(e.podRoot, "net")
}

func (e *podEnv) setupNets(nets []activeNet, noDNS bool) error {
	err := os.MkdirAll(e.netDir(), 0755)
	if err != nil {
		return err
	}

	i := 0
	defer func() {
		if err != nil {
			e.teardownNets(nets[:i])
		}
	}()

	n := activeNet{}

	// did stage0 already make /etc/rkt-resolv.conf (i.e. --dns passed)
	resolvPath := filepath.Join(common.Stage1RootfsPath(e.podRoot), "etc/rkt-resolv.conf")
	_, err = os.Stat(resolvPath)
	if err != nil && !os.IsNotExist(err) {
		return errwrap.Wrap(fmt.Errorf("error statting /etc/rkt-resolv.conf"), err)
	}
	podHasResolvConf := err == nil

	for i, n = range nets {
		if debuglog {
			stderr.Printf("loading network %v with type %v", n.conf.Name, n.conf.Type)
		}

		n.runtime.IfName = fmt.Sprintf(IfNamePattern, i)
		if n.runtime.ConfPath, err = copyFileToDir(n.runtime.ConfPath, e.netDir()); err != nil {
			return errwrap.Wrap(fmt.Errorf("error copying %q to %q", n.runtime.ConfPath, e.netDir()), err)
		}

		// Actually shell out to the plugin
		err = e.netPluginAdd(&n, e.podNS.Path())
		if err != nil {
			return errwrap.Wrap(fmt.Errorf("error adding network %q", n.conf.Name), err)
		}

		// Generate rkt-resolv.conf if it's not already there.
		// The first network plugin that supplies a non-empty
		// DNS response will win, unless noDNS is true (--dns passed to rkt run)
		if !common.IsDNSZero(&n.runtime.DNS) && !noDNS {
			if !podHasResolvConf {
				err := ioutil.WriteFile(
					resolvPath,
					[]byte(common.MakeResolvConf(n.runtime.DNS, "Generated by rkt from network "+n.conf.Name)),
					0644)
				if err != nil {
					return errwrap.Wrap(fmt.Errorf("error creating resolv.conf"), err)
				}
				podHasResolvConf = true
			} else {
				stderr.Printf("Warning: network %v plugin specified DNS configuration, but DNS already supplied", n.conf.Name)
			}
		}
	}
	return nil
}

func (e *podEnv) teardownNets(nets []activeNet) {
	for i := len(nets) - 1; i >= 0; i-- {
		if debuglog {
			stderr.Printf("teardown - executing net-plugin %v", nets[i].conf.Type)
		}

		podNSpath := ""
		if e.podNS != nil {
			podNSpath = e.podNS.Path()
		}

		err := e.netPluginDel(&nets[i], podNSpath)
		if err != nil {
			stderr.PrintE(fmt.Sprintf("error deleting %q", nets[i].conf.Name), err)
		}

		// Delete the conf file to signal that the network was
		// torn down (or at least attempted to)
		if err = os.Remove(nets[i].runtime.ConfPath); err != nil {
			stderr.PrintE(fmt.Sprintf("error deleting %q", nets[i].runtime.ConfPath), err)
		}
	}
}

// netLoader loads network definitions hierarchically. Child loaders can override definitions from parents.
type netLoader struct {
	parent     *netLoader
	configPath string
}

// Build a Loader that looks first for custom (user provided) plugins, then builtin.
func (e *podEnv) newNetLoader() *netLoader {
	return &netLoader{
		parent: &netLoader{
			configPath: path.Join(common.Stage1RootfsPath(e.podRoot), BuiltinNetPath),
		},
		configPath: filepath.Join(e.localConfig, UserNetPathSuffix),
	}
}

func (l *netLoader) loadNets(netsLoadList common.NetList) (map[string]activeNet, error) {
	var parentNets map[string]activeNet
	if l.parent != nil {
		var err error
		if parentNets, err = l.parent.loadNets(netsLoadList); err != nil {
			return nil, err
		}
	} else {
		parentNets = make(map[string]activeNet)
	}

	if debuglog {
		stderr.Printf("loading networks from %v", l.configPath)
	}

	files, err := listFiles(l.configPath)
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	nets := make(map[string]activeNet, len(files))

	for _, filename := range files {
		filepath := filepath.Join(l.configPath, filename)

		if !strings.HasSuffix(filepath, ".conf") {
			continue
		}

		n, err := loadNet(filepath)
		if err != nil {
			return nil, err
		}

		if !(netsLoadList.All() || netsLoadList.Specific(n.conf.Name)) {
			continue
		}

		if _, ok := nets[n.conf.Name]; ok {
			stderr.Printf("%q network already defined, ignoring %v", n.conf.Name, filename)
			continue
		}

		nets[n.conf.Name] = *n
	}

	// merge with parent
	for name, net := range nets {
		if _, exists := parentNets[name]; exists {
			stderr.Printf(`overriding %q network with one from %v`, name, l.configPath)
		}
		parentNets[name] = net
	}
	return parentNets, nil
}

func missingNets(defined common.NetList, loaded []activeNet) []string {
	diff := make(map[string]struct{})
	for _, n := range defined.StringsOnlyNames() {
		if n != "all" {
			diff[n] = struct{}{}
		}
	}

	for _, an := range loaded {
		delete(diff, an.conf.Name)
	}

	var missing []string
	for n := range diff {
		missing = append(missing, n)
	}
	return missing
}

func listFiles(dir string) ([]string, error) {
	dirents, err := ioutil.ReadDir(dir)
	switch {
	case err == nil:
	case os.IsNotExist(err):
		return nil, nil
	default:
		return nil, err
	}

	var files []string
	for _, dent := range dirents {
		if dent.IsDir() {
			continue
		}

		files = append(files, dent.Name())
	}

	return files, nil
}

func loadNet(filepath string) (*activeNet, error) {
	bytes, err := ioutil.ReadFile(filepath)
	if err != nil {
		return nil, err
	}

	n := &NetConf{}
	if err = json.Unmarshal(bytes, n); err != nil {
		return nil, errwrap.Wrap(fmt.Errorf("error loading %v", filepath), err)
	}

	return &activeNet{
		confBytes: bytes,
		conf:      n,
		runtime: &netinfo.NetInfo{
			NetName:  n.Name,
			ConfPath: filepath,
		},
	}, nil
}

func copyFileToDir(src, dstdir string) (string, error) {
	dst := filepath.Join(dstdir, filepath.Base(src))

	s, err := os.Open(src)
	if err != nil {
		return "", err
	}
	defer s.Close()

	d, err := os.Create(dst)
	if err != nil {
		return "", err
	}
	defer d.Close()

	_, err = io.Copy(d, s)
	return dst, err
}
