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
	"net"
	"os"
	"runtime"

	"github.com/coreos/rkt/Godeps/_workspace/src/github.com/vishvananda/netlink"

	"github.com/coreos/rkt/networking/ipam"
	rktnet "github.com/coreos/rkt/networking/net"
	"github.com/coreos/rkt/networking/util"
)

func init() {
	// this ensures that main runs only on main thread (thread group leader).
	// since namespace ops (unshare, setns) are done for a single thread, we
	// must ensure that the goroutine does not jump from OS thread to thread
	runtime.LockOSThread()
}

type Net struct {
	rktnet.Net
	IPMasq bool `json:"ipMasq"`
	MTU    int  `json:"mtu"`
}

func setupPodVeth(podID, netns, ifName string, mtu int, ipConf *ipam.IPConfig) (string, error) {
	var hostVethName string
	err := util.WithNetNSPath(netns, func(hostNS *os.File) error {
		entropy := podID + ifName

		hostVeth, _, err := util.SetupVeth(entropy, ifName, mtu, hostNS)
		if err != nil {
			return err
		}

		err = ipam.ApplyIPConfig(ifName, ipConf)
		if err != nil {
			return err
		}

		hostVethName = hostVeth.Attrs().Name

		return nil
	})
	return hostVethName, err
}

func setupHostVeth(vethName string, ipConf *ipam.IPConfig) error {
	// hostVeth moved namespaces and may have a new ifindex
	veth, err := netlink.LinkByName(vethName)
	if err != nil {
		return fmt.Errorf("failed to lookup %q: %v", vethName, err)
	}

	// TODO(eyakubovich): IPv6
	ipn := &net.IPNet{
		IP:   ipConf.Gateway,
		Mask: net.CIDRMask(31, 32),
	}
	addr := &netlink.Addr{IPNet: ipn, Label: ""}
	if err = netlink.AddrAdd(veth, addr); err != nil {
		return fmt.Errorf("failed to add IP addr to veth: %v", err)
	}

	// dst happens to be the same as IP/net of host veth
	if err = util.AddHostRoute(ipn, nil, veth); err != nil && !os.IsExist(err) {
		return fmt.Errorf("failed to add route on host: %v", err)
	}

	return nil
}

func cmdAdd(args *util.CmdArgs) error {
	conf := Net{}
	if err := rktnet.LoadNet(args.NetConf, &conf); err != nil {
		return fmt.Errorf("failed to load %q: %v", args.NetConf, err)
	}

	// run the IPAM plugin and get back the config to apply
	ipConf, err := ipam.ExecPluginAdd(conf.IPAM.Type)
	if err != nil {
		return err
	}

	hostVethName, err := setupPodVeth(args.PodID.String(), args.Netns, args.IfName, conf.MTU, ipConf)
	if err != nil {
		return err
	}

	if err = setupHostVeth(hostVethName, ipConf); err != nil {
		return err
	}

	if conf.IPMasq {
		chain := fmt.Sprintf("RKT-%s-%s", conf.Name, args.PodID.String()[:8])
		if err = util.SetupIPMasq(ipConf.IP, chain); err != nil {
			return err
		}
	}

	return rktnet.PrintIfConfig(&rktnet.IfConfig{
		IP:     ipConf.IP.IP,
		HostIP: ipConf.Gateway,
	})
}

func cmdDel(args *util.CmdArgs) error {
	conf := Net{}
	if err := rktnet.LoadNet(args.NetConf, &conf); err != nil {
		return fmt.Errorf("failed to load %q: %v", args.NetConf, err)
	}

	var ipn *net.IPNet
	err := util.WithNetNSPath(args.Netns, func(hostNS *os.File) error {
		var err error
		ipn, err = util.DelLinkByNameAddr(args.IfName, netlink.FAMILY_V4)
		return err
	})
	if err != nil {
		return err
	}

	if conf.IPMasq {
		chain := fmt.Sprintf("RKT-%s-%s", conf.Name, args.PodID.String()[:8])
		if err = util.TeardownIPMasq(ipn, chain); err != nil {
			return err
		}
	}

	return ipam.ExecPluginDel(conf.IPAM.Type)
}

func main() {
	util.PluginMain(cmdAdd, cmdDel)
}
