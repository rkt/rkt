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

package util

import (
	"log"
	"os"

	"github.com/coreos/rkt/Godeps/_workspace/src/github.com/appc/spec/schema/types"
)

type CmdArgs struct {
	PodID    types.UUID
	Netns    string
	IfName   string
	NetConf  string
	NetName  string
	IPAMPath string
}

func PluginMain(cmdAdd, cmdDel func(_ *CmdArgs) error) {
	var cmd, podID, netns, ifName, netConf, netName, ipamPath string

	vars := []struct {
		name string
		val  *string
	}{
		{"RKT_NETPLUGIN_COMMAND", &cmd},
		{"RKT_NETPLUGIN_PODID", &podID},
		{"RKT_NETPLUGIN_NETNS", &netns},
		{"RKT_NETPLUGIN_IFNAME", &ifName},
		{"RKT_NETPLUGIN_NETCONF", &netConf},
		{"RKT_NETPLUGIN_NETNAME", &netName},
		{"RKT_NETPLUGIN_IPAMPATH", &ipamPath},
	}

	argsMissing := false
	for _, v := range vars {
		*v.val = os.Getenv(v.name)
		if *v.val == "" {
			log.Printf("%v env variable missing", v.name)
			argsMissing = true
		}
	}

	if argsMissing {
		os.Exit(1)
	}

	pid, err := types.NewUUID(podID)
	if err != nil {
		log.Print("Error parsing Pod ID (%v): %v", podID, err)
		os.Exit(1)
	}

	args := &CmdArgs{
		PodID:    *pid,
		Netns:    netns,
		IfName:   ifName,
		NetConf:  netConf,
		NetName:  netName,
		IPAMPath: ipamPath,
	}

	switch cmd {
	case "ADD":
		err = cmdAdd(args)

	case "DEL":
		err = cmdDel(args)

	default:
		log.Printf("Unknown RKT_NETPLUGIN_COMMAND: %v", cmd)
		os.Exit(1)
	}

	if err != nil {
		log.Printf("%v: %v", cmd, err)
		os.Exit(1)
	}
}
