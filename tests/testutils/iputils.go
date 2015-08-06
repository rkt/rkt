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

package testutils

import (
	"fmt"
	"net"
	"strings"

	"github.com/coreos/rkt/Godeps/_workspace/src/github.com/vishvananda/netlink"
)

func getDefaultGW(family int) (string, error) {
	l, err := netlink.LinkByName("lo")
	if err != nil {
		return "", err
	}

	routes, err := netlink.RouteList(l, family)
	if err != nil {
		return "", err
	}

	return routes[0].Gw.String(), nil
}
func GetDefaultGWv4() (string, error) {
	return getDefaultGW(netlink.FAMILY_V4)
}

func GetDefaultGWv6() (string, error) {
	return getDefaultGW(netlink.FAMILY_V6)
}

func GetIPs(ifaceWanted string, familyWanted int) ([]string, error) {
	ips := make([]string, 0)
	ifaces, err := net.Interfaces()
	if err != nil {
		return ips, err
	}
	for _, iface := range ifaces {
		if iface.Name != ifaceWanted {
			continue
		}

		addrs, _ := iface.Addrs()
		for _, addr := range addrs {
			addrString := addr.String()
			ip, _, err := net.ParseCIDR(addrString)
			if err != nil {
				return ips, err
			}

			if strings.Contains(addrString, ".") && familyWanted == netlink.FAMILY_V4 ||
				strings.Contains(addrString, ":") && familyWanted == netlink.FAMILY_V6 {
				ips = append(ips, ip.String())
			}
		}
	}
	return ips, err
}

func GetIPsv4(iface string) ([]string, error) {
	return GetIPs(iface, netlink.FAMILY_V4)
}
func GetIPsv6(iface string) ([]string, error) {
	return GetIPs(iface, netlink.FAMILY_V6)
}

func GetGW(iface string, family int) (string, error) {
	return "", fmt.Errorf("Not implemented")
}
func GetGWv4(iface string) (string, error) {
	return GetGW(iface, netlink.FAMILY_V4)
}

func GetGWv6(iface string) (string, error) {
	return GetGW(iface, netlink.FAMILY_V4)
}
