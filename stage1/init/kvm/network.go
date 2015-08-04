package kvm

import (
	"fmt"

	"github.com/coreos/rkt/networking"
)

// KVMNetParams returns additionall parameters that need to be passed to kernel and lkvm tool to configure networks properly
func GetKVMNetParams(network networking.Networking) ([]string, []string, error) {
	lkvmParams := []string{}
	kernelParams := []string{}

	for i, netParams := range network.GetNetworkParameters() {
		//

		// https://www.kernel.org/doc/Documentation/filesystems/nfs/nfsroot.txt
		// ip=<client-ip>:<server-ip>:<gw-ip>:<netmask>:<hostname>:<device>:<autoconf>:<dns0-ip>:<dns1-ip>
		var gw string
		if netParams.IPMasq {
			gw = netParams.HostIP.String()
		}
		kernelParams = append(kernelParams, "ip="+netParams.GuestIP.String()+"::"+gw+":"+netParams.Mask.String()+":"+fmt.Sprintf(networking.IfNamePattern, i)+":::")

		lkvmParams = append(lkvmParams, "--network", "mode=tap,tapif="+netParams.IfName+",host_ip="+netParams.HostIP.String()+",guest_ip="+netParams.GuestIP.String())
	}

	return lkvmParams, kernelParams, nil
}
