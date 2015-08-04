// kvm.go file provides networking supporting functions for kvm flavor
package networking

import (
	"encoding/json"
	"fmt"
	"log"
	"net"

	"github.com/coreos/rkt/Godeps/_workspace/src/github.com/appc/cni/pkg/ip"
	"github.com/coreos/rkt/Godeps/_workspace/src/github.com/appc/cni/pkg/plugin"
	"github.com/coreos/rkt/Godeps/_workspace/src/github.com/appc/spec/schema/types"
	"github.com/coreos/rkt/Godeps/_workspace/src/github.com/vishvananda/netlink"

	"github.com/coreos/rkt/common"
	"github.com/coreos/rkt/networking/tuntap"
)

// setupTapDevice creates persistent tap devices
// and returns a newly created netlink.Link structure
func setupTapDevice() (netlink.Link, error) {
	ifName, err := tuntap.CreatePersistentIface(tuntap.Tap)
	if err != nil {
		return nil, fmt.Errorf("tuntap persist %v", err)
	}
	link, err := netlink.LinkByName(ifName)
	if err != nil {
		return nil, fmt.Errorf("cannot find link %q: %v", ifName, err)
	}
	err = netlink.LinkSetUp(link)
	if err != nil {
		return nil, fmt.Errorf("cannot set link up %q: %v", ifName, err)
	}
	return link, nil
}

func kvmSetupNetAddressing(network *Networking, n activeNet, ifName string) error {
	// TODO: very ugly hack, that go through upper plugin, down to ipam plugin
	n.conf.Type = n.conf.IPAM.Type
	output, err := network.execNetPlugin("ADD", &n, ifName)
	if err != nil {
		return fmt.Errorf("problem executing network plugin %q (%q): %v", n.conf.Type, ifName, err)
	}

	result := plugin.Result{}
	if err = json.Unmarshal(output, &result); err != nil {
		return fmt.Errorf("error parsing %q result: %v", n.conf.Name, err)
	}

	if result.IP4 == nil {
		return fmt.Errorf("net-plugin returned no IPv4 configuration")
	}

	n.runtime.IP, n.runtime.Mask, n.runtime.HostIP = result.IP4.IP.IP, net.IP(result.IP4.IP.Mask), result.IP4.Gateway
	return nil
}

func kvmSetup(podRoot string, podID types.UUID, fps []ForwardedPort, privateNetList common.PrivateNetList, localConfig string) (*Networking, error) {
	network := Networking{
		podEnv: podEnv{
			podRoot:      podRoot,
			podID:        podID,
			netsLoadList: privateNetList,
			localConfig:  localConfig,
		},
	}
	var e error
	network.nets, e = network.loadNets()
	if e != nil {
		return nil, fmt.Errorf("error loading network definitions: %v", e)
	}

	for _, n := range network.nets {
		switch n.conf.Type {
		case "ptp":
			link, err := setupTapDevice()
			if err != nil {
				return nil, err
			}
			ifName := link.Attrs().Name
			n.runtime.IfName = ifName

			err = kvmSetupNetAddressing(&network, n, ifName)
			if err != nil {
				return nil, err
			}

			// add address to host tap device
			err = netlink.AddrAdd(
				link,
				&netlink.Addr{
					IPNet: &net.IPNet{
						IP:   n.runtime.HostIP,
						Mask: net.IPMask(n.runtime.Mask),
					},
					Label: ifName,
				})
			if err != nil {
				return nil, fmt.Errorf("cannot add address to host tap device %q: %v", ifName, err)
			}

			if n.conf.IPMasq {
				chain := "CNI-" + n.conf.Name
				if err = ip.SetupIPMasq(&net.IPNet{
					IP:   n.runtime.IP,
					Mask: net.IPMask(n.runtime.Mask),
				}, chain); err != nil {
					return nil, err
				}
			}
		default:
			return nil, fmt.Errorf("network %q have unsupported type: %q", n.conf.Name, n.conf.Type)
		}
	}
	err := network.forwardPorts(fps, network.GetDefaultIP())
	if err != nil {
		return nil, err
	}

	return &network, nil
}

func (e *Networking) teardownKvmNets(nets []activeNet) {
	for _, n := range nets {
		switch n.conf.Type {
		case "ptp":
			tuntap.RemovePersistentIface(n.runtime.IfName, tuntap.Tap)
			n.conf.Type = n.conf.IPAM.Type

			_, err := e.execNetPlugin("DEL", &n, n.runtime.IfName)
			if err != nil {
				log.Printf("Error executing network plugin: %q", err)
			}
		default:
			log.Printf("Unsupported network type: %q", n.conf.Type)
		}
	}
}

// NetParams exposes conf(NetConf)/runtime(NetInfo) data to stage1/init client
type NetParams struct {
	// runtime based information
	HostIP  net.IP
	GuestIP net.IP
	Mask    net.IP
	IfName  string
	// TODO: required for other type of plugins, not yet available because what networking.Networking stores
	// Net net.IPNet

	// configuration based information
	Name   string
	Type   string
	IPMasq bool
}

// GetNetworkParameters returns network parameters created
// by plugins, which are required for stage1 executor to run (only for KVM)
func (e *Networking) GetNetworkParameters() []NetParams {
	np := []NetParams{}
	_ = np
	for _, an := range e.nets {
		np = append(np, NetParams{
			HostIP:  an.runtime.HostIP,
			GuestIP: an.runtime.IP,
			IfName:  an.runtime.IfName,
			Mask:    an.runtime.Mask,
			// Net: // TODO: from where
			Name:   an.conf.Name,
			Type:   an.conf.Type,
			IPMasq: an.conf.IPMasq,
		})
	}

	return np
}
