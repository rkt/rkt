// Copyright 2015 CoreOS, Inc.
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
	"fmt"
	"os"
	"runtime"

	"github.com/coreos/rkt/Godeps/_workspace/src/github.com/vishvananda/netlink"

	"github.com/coreos/rkt/networking/ipam"
	rktnet "github.com/coreos/rkt/networking/net"
	"github.com/coreos/rkt/networking/util"
)

type Net struct {
	rktnet.Net
	Master string `json:"master"`
	Mode   string `json:"mode"`
	IPMasq bool   `json:"ipMasq"`
	MTU    int    `json:"mtu"`
}

func init() {
	// this ensures that main runs only on main thread (thread group leader).
	// since namespace ops (unshare, setns) are done for a single thread, we
	// must ensure that the goroutine does not jump from OS thread to thread
	runtime.LockOSThread()
}

func loadConf(path string) (*Net, error) {
	n := &Net{}
	if err := rktnet.LoadNet(path, n); err != nil {
		return nil, fmt.Errorf("failed to load %q: %v", path, err)
	}
	if n.Master == "" {
		return nil, fmt.Errorf(`"master" field is required. It specifies the host interface name to virtualize`)
	}
	return n, nil
}

func modeFromString(s string) (netlink.MacvlanMode, error) {
	switch s {
	case "":
		return netlink.MACVLAN_MODE_BRIDGE, nil
	case "private":
		return netlink.MACVLAN_MODE_PRIVATE, nil
	case "vepa":
		return netlink.MACVLAN_MODE_VEPA, nil
	case "bridge":
		return netlink.MACVLAN_MODE_BRIDGE, nil
	case "passthru":
		return netlink.MACVLAN_MODE_PASSTHRU, nil
	default:
		return 0, fmt.Errorf("unknown macvlan mode: %q", s)
	}
}

func createMacvlan(conf *Net, ifName string, netns *os.File) error {
	mode, err := modeFromString(conf.Mode)
	if err != nil {
		return err
	}

	m, err := netlink.LinkByName(conf.Master)
	if err != nil {
		return fmt.Errorf("failed to lookup master %q: %v", conf.Master, err)
	}

	mv := &netlink.Macvlan{
		LinkAttrs: netlink.LinkAttrs{
			MTU:         conf.MTU,
			Name:        ifName,
			ParentIndex: m.Attrs().Index,
			Namespace:   netlink.NsFd(int(netns.Fd())),
		},
		Mode: mode,
	}

	if err := netlink.LinkAdd(mv); err != nil {
		return fmt.Errorf("failed to create macvlan: %v", err)
	}

	return err
}

func cmdAdd(args *util.CmdArgs) error {
	n, err := loadConf(args.NetConf)
	if err != nil {
		return err
	}

	netns, err := os.Open(args.Netns)
	if err != nil {
		return fmt.Errorf("failed to open netns %q: %v", netns, err)
	}
	defer netns.Close()

	tmpName := "veth" + args.PodID.String()[:4]
	if err = createMacvlan(n, tmpName, netns); err != nil {
		return err
	}

	// run the IPAM plugin and get back the config to apply
	ipConf, err := ipam.ExecPluginAdd(n.Net.IPAM.Type)
	if err != nil {
		return err
	}

	err = util.WithNetNS(netns, func(_ *os.File) error {
		err := renameLink(tmpName, args.IfName)
		if err != nil {
			return fmt.Errorf("failed to rename macvlan to %q: %v", args.IfName, err)
		}

		return ipam.ApplyIPConfig(args.IfName, ipConf)
	})
	if err != nil {
		return err
	}

	if n.IPMasq {
		chain := "RKT-" + n.Name
		if err = util.SetupIPMasq(util.Network(ipConf.IP), chain); err != nil {
			return err
		}
	}

	return rktnet.PrintIfConfig(&rktnet.IfConfig{
		IP: ipConf.IP.IP,
	})
}

func cmdDel(args *util.CmdArgs) error {
	n, err := loadConf(args.NetConf)
	if err != nil {
		return err
	}

	err = ipam.ExecPluginDel(n.Net.IPAM.Type)
	if err != nil {
		return err
	}

	return util.WithNetNSPath(args.Netns, func(hostNS *os.File) error {
		return util.DelLinkByName(args.IfName)
	})
}

func renameLink(curName, newName string) error {
	link, err := netlink.LinkByName(curName)
	if err != nil {
		return err
	}

	return netlink.LinkSetName(link, newName)
}

func main() {
	util.PluginMain(cmdAdd, cmdDel)
}
