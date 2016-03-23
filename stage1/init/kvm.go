// Copyright 2014 The rkt Authors
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

//+build linux

package main

import (
	"errors"
	"path/filepath"

	"github.com/coreos/rkt/common"
	"github.com/coreos/rkt/networking"
	stage1commontypes "github.com/coreos/rkt/stage1/common/types"
	stage1initcommon "github.com/coreos/rkt/stage1/init/common"
	"github.com/coreos/rkt/stage1/init/kvm"
	"github.com/hashicorp/errwrap"
)

// KvmNetworkingToSystemd generates systemd unit files for a pod according to network configuration
func KvmNetworkingToSystemd(p *stage1commontypes.Pod, n *networking.Networking) error {
	podRoot := common.Stage1RootfsPath(p.Root)

	// networking
	netDescriptions := kvm.GetNetworkDescriptions(n)
	if err := kvm.GenerateNetworkInterfaceUnits(filepath.Join(podRoot, stage1initcommon.UnitsDir), netDescriptions); err != nil {
		return errwrap.Wrap(errors.New("failed to transform networking to units"), err)
	}

	return nil
}
