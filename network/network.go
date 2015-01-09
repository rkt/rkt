package network

import (
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"syscall"

	"github.com/appc/spec/schema/types"
	"github.com/coreos/rocket/Godeps/_workspace/src/github.com/vishvananda/netlink"

	"github.com/coreos/rocket/network/util"
)

const (
	ifnamePattern = "eth%d"
	selfNetNS = "/proc/self/ns/net"
)

type activeNet struct {
	Net
	ifName string
	ipn    *net.IPNet
}

type Network struct {
	MetadataIP net.IP

	contID     types.UUID
	hostNS     *os.File
	contNS     *os.File
	contNSPath string
	nets       []activeNet
	plugins    map[string]*NetPlugin
}

func Setup(contID types.UUID) (*Network, error) {
	var err error
	n := Network{contID: contID}

	defer func() {
		// cleanup on error
		if err != nil {
			n.Teardown()
		}
	}()

	if n.hostNS, n.contNS, err = basicNetNS(); err != nil {
		return nil, err
	}
	// we're in contNS!

	contNSPath := filepath.Join("/var/lib/rkt/containers", contID.String(), "ns")
	if err = bindMountFile(selfNetNS, contNSPath, "net"); err != nil {
		return nil, err
	}
	n.contNSPath = filepath.Join(contNSPath, "net")

	n.plugins, err = LoadNetPlugins()
	if err != nil {
		return nil, fmt.Errorf("error loading plugin definitions: %v", err)
	}

	nets, err := LoadNets()
	if err != nil {
		return nil, fmt.Errorf("error loading network definitions: %v", err)
	}

	err = withNetNS(n.contNS, n.hostNS, func() error {
		n.nets, err = setupNets(contID, n.contNSPath, n.plugins, nets)
		return err
	})
	if err != nil {
		return nil, err
	}

	// last net is the default
	n.MetadataIP = n.nets[len(n.nets)-1].ipn.IP

	return &n, nil
}

func (n *Network) Teardown() {
	// teardown everything in reverse order of setup.
	// this is called during error case as well so not
	// everything maybe setup.
	// N.B. better to keep going in case of errors to get as much
	// cleaned up as possible

	if n.contNS == nil || n.hostNS == nil {
		return
	}

	if err := n.EnterHostNS(); err != nil {
		log.Print(err)
		return
	}

	teardownNets(n.contID, n.contNSPath, n.plugins, n.nets)

	if n.contNSPath == "" {
		return
	}

	if err := syscall.Unmount(n.contNSPath, 0); err != nil {
		log.Print("Error unmounting %q: %v", n.contNSPath, err)
	}
}

// sets up new netns with just lo
func basicNetNS() (hostNS, contNS *os.File, err error) {
	hostNS, contNS, err = newNetNS()
	if err != nil {
		err = fmt.Errorf("failed to create new netns: %v", err)
		return
	}
	// we're in contNS!!

	if err = loUp(); err != nil {
		hostNS.Close()
		contNS.Close()
		return nil, nil, err
	}

	return
}


func (n *Network) EnterHostNS() error {
	return util.SetNS(n.hostNS, syscall.CLONE_NEWNET)
}

func (n *Network) EnterContNS() error {
	return util.SetNS(n.contNS, syscall.CLONE_NEWNET)
}

func setupNets(contID types.UUID, netns string, plugins map[string]*NetPlugin, nets []Net) ([]activeNet, error) {
	var err error

	active := []activeNet{}

	for i, nt := range nets {
		plugin, ok := plugins[nt.Type]
		if !ok {
			err = fmt.Errorf("could not find network plugin %q\n", nt.Type)
			break
		}

		an := activeNet{
			Net: nt,
			ifName: fmt.Sprintf(ifnamePattern, i),
		}

		log.Printf("Executing net-plugin %v", nt.Type)

		an.ipn, err = plugin.Add(&nt, contID.String(), netns, nt.args, an.ifName)
		if err != nil {
			err = fmt.Errorf("error adding network %q: %v\n", nt.Name, err)
			break
		}

		active = append(active, an)
	}

	log.Print("Done executing net plugins")

	if err != nil {
		teardownNets(contID, netns, plugins, active)
		return nil, err
	}

	return active, nil
}

func teardownNets(contID types.UUID, netns string, plugins map[string]*NetPlugin, nets []activeNet) {
	for i := len(nets) - 1; i >= 0; i-- {
		nt := nets[i]
		plugin := plugins[nt.Type]

		err := plugin.Del(&nt.Net, contID.String(), netns, nt.args, nt.ifName)
		if err != nil {
			log.Printf("Error deleting %q: %v", nt.Name, err)
		}
	}
}

func newNetNS() (hostNS, childNS *os.File, err error) {
	defer func() {
		if err != nil {
			if hostNS != nil {
				hostNS.Close()
			}
			if childNS != nil {
				childNS.Close()
			}
		}
	}()

	hostNS, err = os.Open(selfNetNS)
	if err != nil {
		return
	}

	if err = syscall.Unshare(syscall.CLONE_NEWNET); err != nil {
		return
	}

	childNS, err = os.Open(selfNetNS)
	if err != nil {
		util.SetNS(hostNS, syscall.CLONE_NEWNET)
		return
	}

	return
}

// execute f() in tgtNS
func withNetNS(curNS, tgtNS *os.File, f func() error) error {
	if err := util.SetNS(tgtNS, syscall.CLONE_NEWNET); err != nil {
		return err
	}

	if err := f(); err != nil {
		return err
	}

	return util.SetNS(curNS, syscall.CLONE_NEWNET)
}

func loUp() error {
	lo, err := netlink.LinkByName("lo")
	if err != nil {
		return fmt.Errorf("failed to lookup lo: %v", err)
	}

	if err := netlink.LinkSetUp(lo); err != nil {
		return fmt.Errorf("failed to set lo up: %v", err)
	}

	return nil
}

func bindMountFile(src, dstDir, dstFile string) error {
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return err
	}

	dst := filepath.Join(dstDir, dstFile)

	// mount point has to be an existing file
	f, err := os.Create(dst)
	if err != nil {
		return err
	}
	f.Close()

	return syscall.Mount(src, dst, "none", syscall.MS_BIND, "")
}
