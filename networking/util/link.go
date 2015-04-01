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

package util

import (
	"crypto/sha512"
	"fmt"
	"net"
	"os"

	"github.com/coreos/rkt/Godeps/_workspace/src/github.com/vishvananda/netlink"
)

func makeVeth(name, peer string, mtu int) (netlink.Link, error) {
	veth := &netlink.Veth{
		LinkAttrs: netlink.LinkAttrs{
			Name:  name,
			Flags: net.FlagUp,
			MTU:   mtu,
		},
		PeerName: peer,
	}
	if err := netlink.LinkAdd(veth); err != nil {
		return nil, err
	}

	return veth, nil
}

func hash(s string) string {
	h := sha512.New()
	h.Write([]byte(s))
	return fmt.Sprintf("%x", h.Sum(nil))
}

// SetupVeth sets up a virtual ethernet link.
// Should be in pod netns.
// TODO(eyakubovich): get rid of entropy and ask kernel to pick name via pattern
func SetupVeth(entropy, podVethName string, mtu int, hostNS *os.File) (hostVeth, podVeth netlink.Link, err error) {
	// NetworkManager (recent versions) will ignore veth devices that start with "veth"
	hostVethName := "veth" + hash(entropy)[:4]
	hostVeth, err = makeVeth(hostVethName, podVethName, mtu)
	if err != nil {
		err = fmt.Errorf("failed to make veth pair: %v", err)
		return
	}

	if err = netlink.LinkSetUp(hostVeth); err != nil {
		err = fmt.Errorf("failed to set %q up: %v", hostVethName, err)
		return
	}

	podVeth, err = netlink.LinkByName(podVethName)
	if err != nil {
		err = fmt.Errorf("failed to lookup %q: %v", podVethName, err)
		return
	}

	if err = netlink.LinkSetUp(podVeth); err != nil {
		err = fmt.Errorf("failed to set %q up: %v", podVethName, err)
		return
	}

	if err = netlink.LinkSetNsFd(hostVeth, int(hostNS.Fd())); err != nil {
		err = fmt.Errorf("failed to move veth to host netns: %v", err)
		return
	}

	return
}

// DelLinkByName removes an interface link.
func DelLinkByName(ifName string) error {
	iface, err := netlink.LinkByName(ifName)
	if err != nil {
		return fmt.Errorf("failed to lookup %q: %v", ifName, err)
	}

	if err = netlink.LinkDel(iface); err != nil {
		return fmt.Errorf("failed to delete %q: %v", ifName, err)
	}

	return nil
}

// DelLinkByNameAddr remove an interface returns its IP address
// of the specified family
func DelLinkByNameAddr(ifName string, family int) (*net.IPNet, error) {
	iface, err := netlink.LinkByName(ifName)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup %q: %v", ifName, err)
	}

	addrs, err := netlink.AddrList(iface, family)
	if err != nil || len(addrs) == 0 {
		return nil, fmt.Errorf("failed to get IP addresses for %q: %v", ifName, err)
	}

	if err = netlink.LinkDel(iface); err != nil {
		return nil, fmt.Errorf("failed to delete %q: %v", ifName, err)
	}

	return addrs[0].IPNet, nil
}
