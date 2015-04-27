// Copyright 2014 CoreOS, Inc.
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

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"runtime"
	"syscall"

	"github.com/coreos/rkt/Godeps/_workspace/src/github.com/appc/cni/pkg/ip"
	"github.com/coreos/rkt/Godeps/_workspace/src/github.com/appc/cni/pkg/ns"
	"github.com/coreos/rkt/Godeps/_workspace/src/github.com/appc/cni/pkg/plugin"
	"github.com/coreos/rkt/Godeps/_workspace/src/github.com/appc/cni/pkg/skel"
	"github.com/coreos/rkt/Godeps/_workspace/src/github.com/vishvananda/netlink"
)

const defaultBrName = "cni0"

type NetConf struct {
	plugin.NetConf
	BrName string `json:"bridge"`
	IsGW   bool   `json:"isGateway"`
	IPMasq bool   `json:"ipMasq"`
	MTU    int    `json:"mtu"`
}

func init() {
	// this ensures that main runs only on main thread (thread group leader).
	// since namespace ops (unshare, setns) are done for a single thread, we
	// must ensure that the goroutine does not jump from OS thread to thread
	runtime.LockOSThread()
}

func loadNetConf(bytes []byte) (*NetConf, error) {
	n := &NetConf{
		BrName: defaultBrName,
	}
	if err := json.Unmarshal(bytes, n); err != nil {
		return nil, fmt.Errorf("failed to load netconf: %v", err)
	}
	return n, nil
}

func ensureBridgeAddr(br *netlink.Bridge, ipn *net.IPNet) error {
	addrs, err := netlink.AddrList(br, syscall.AF_INET)
	if err != nil && err != syscall.ENOENT {
		return fmt.Errorf("could not get list of IP addresses: %v", err)
	}

	// if there're no addresses on the bridge, it's ok -- we'll add one
	if len(addrs) > 0 {
		ipnStr := ipn.String()
		for _, a := range addrs {
			// string comp is actually easiest for doing IPNet comps
			if a.IPNet.String() == ipnStr {
				return nil
			}
		}
		return fmt.Errorf("%q already has an IP address different from %v", br.Name, ipn.String())
	}

	addr := &netlink.Addr{IPNet: ipn, Label: ""}
	if err := netlink.AddrAdd(br, addr); err != nil {
		return fmt.Errorf("could not add IP address to %q: %v", br.Name, err)
	}
	return nil
}

func bridgeByName(name string) (*netlink.Bridge, error) {
	l, err := netlink.LinkByName(name)
	if err != nil {
		return nil, fmt.Errorf("could not lookup %q: %v", name, err)
	}
	br, ok := l.(*netlink.Bridge)
	if !ok {
		return nil, fmt.Errorf("%q already exists but is not a bridge", name)
	}
	return br, nil
}

func ensureBridge(brName string, mtu int, ipn *net.IPNet) (*netlink.Bridge, error) {
	br := &netlink.Bridge{
		LinkAttrs: netlink.LinkAttrs{
			Name: brName,
			MTU:  mtu,
		},
	}

	if err := netlink.LinkAdd(br); err != nil {
		if err != syscall.EEXIST {
			return nil, fmt.Errorf("could not add %q: %v", brName, err)
		}

		// it's ok if the device already exists as long as config is similar
		br, err = bridgeByName(brName)
		if err != nil {
			return nil, err
		}
	}

	if err := netlink.LinkSetUp(br); err != nil {
		return nil, err
	}

	if ipn != nil {
		return br, ensureBridgeAddr(br, ipn)
	}

	return br, nil
}

func setupVeth(netns string, br *netlink.Bridge, ifName string, mtu int, pr *plugin.Result) error {
	var hostVethName string

	err := ns.WithNetNSPath(netns, func(hostNS *os.File) error {
		// create the veth pair in the container and move host end into host netns
		hostVeth, _, err := ip.SetupVeth(netns, ifName, mtu, hostNS)
		if err != nil {
			return err
		}

		if err = plugin.ConfigureIface(ifName, pr); err != nil {
			return err
		}

		hostVethName = hostVeth.Attrs().Name
		return nil
	})
	if err != nil {
		return err
	}

	// need to lookup hostVeth again as its index has changed during ns move
	hostVeth, err := netlink.LinkByName(hostVethName)
	if err != nil {
		return fmt.Errorf("failed to lookup %q: %v", hostVethName, err)
	}

	// connect host veth end to the bridge
	if err = netlink.LinkSetMaster(hostVeth, br); err != nil {
		return fmt.Errorf("failed to connect %q to bridge %v: %v", hostVethName, br.Attrs().Name, err)
	}

	return nil
}

func calcGatewayIP(ipn *net.IPNet) net.IP {
	nid := ipn.IP.Mask(ipn.Mask)
	return ip.NextIP(nid)
}

func setupBridge(n *NetConf, ipConf *plugin.IPConfig) (*netlink.Bridge, error) {
	var gwn *net.IPNet
	if n.IsGW {
		gwn = &net.IPNet{
			IP:   ipConf.Gateway,
			Mask: ipConf.IP.Mask,
		}
	}

	// create bridge if necessary
	br, err := ensureBridge(n.BrName, n.MTU, gwn)
	if err != nil {
		return nil, fmt.Errorf("failed to create bridge %q: %v", n.BrName, err)
	}

	return br, nil
}

func cmdAdd(args *skel.CmdArgs) error {
	n, err := loadNetConf(args.StdinData)
	if err != nil {
		return err
	}

	// run the IPAM plugin and get back the config to apply
	result, err := plugin.ExecAdd(n.IPAM.Type, args.StdinData)
	if err != nil {
		return err
	}

	if result.IP4 == nil {
		return errors.New("IPAM plugin returned missing IPv4 config")
	}

	if result.IP4.Gateway == nil && n.IsGW {
		result.IP4.Gateway = calcGatewayIP(&result.IP4.IP)
	}

	br, err := setupBridge(n, result.IP4)
	if err != nil {
		return err
	}

	if err = setupVeth(args.Netns, br, args.IfName, n.MTU, result); err != nil {
		return err
	}

	if n.IPMasq {
		chain := "CNI-" + n.Name
		if err = ip.SetupIPMasq(ip.Network(&result.IP4.IP), chain); err != nil {
			return err
		}
	}

	return plugin.PrintResult(result)
}

func cmdDel(args *skel.CmdArgs) error {
	n, err := loadNetConf(args.StdinData)
	if err != nil {
		return err
	}

	err = ns.WithNetNSPath(args.Netns, func(hostNS *os.File) error {
		return ip.DelLinkByName(args.IfName)
	})
	if err != nil {
		return err
	}

	return plugin.ExecDel(n.IPAM.Type, args.StdinData)
}

func main() {
	skel.PluginMain(cmdAdd, cmdDel)
}
