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

package kvm

import (
	"fmt"
	"net"

	"github.com/coreos/rkt/networking"
)

// GetNetworkDescriptions explicitly convert slice of activeNets to slice of netDescribers
// which is slice required by GetKVMNetArgs
func GetNetworkDescriptions(n *networking.Networking) []netDescriber {
	var nds []netDescriber
	for _, an := range n.GetActiveNetworks() {
		nds = append(nds, an)
	}
	return nds
}

// netDescriber is something that describes network configuration
type netDescriber interface {
	HostIP() net.IP
	GuestIP() net.IP
	Mask() net.IP
	IfName() string
	IPMasq() bool
}

// GetKVMNetArgs returns additional arguments that need to be passed to kernel
// and lkvm tool to configure networks properly.
// Logic is based on Network configuration extracted from Networking struct
// and essentially from activeNets that expose netDescriber behavior
func GetKVMNetArgs(nds []netDescriber) ([]string, []string, error) {

	var lkvmArgs []string
	var kernelParams []string

	for i, nd := range nds {
		// https://www.kernel.org/doc/Documentation/filesystems/nfs/nfsroot.txt
		// ip=<client-ip>:<server-ip>:<gw-ip>:<netmask>:<hostname>:<device>:<autoconf>:<dns0-ip>:<dns1-ip>
		var gw string
		if nd.IPMasq() {
			gw = nd.HostIP().String()
		}
		kernelParam := fmt.Sprintf("ip=%s::%s:%s::%s:::", nd.GuestIP(), gw, nd.Mask(), fmt.Sprintf(networking.IfNamePattern, i))
		kernelParams = append(kernelParams, kernelParam)

		lkvmArgs = append(lkvmArgs, "--network")
		lkvmArg := fmt.Sprintf("mode=tap,tapif=%s,host_ip=%s,guest_ip=%s", nd.IfName(), nd.HostIP(), nd.GuestIP())
		lkvmArgs = append(lkvmArgs, lkvmArg)
	}

	return lkvmArgs, kernelParams, nil
}
