// Copyright 2017 The rkt Authors
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

package stage1_fly

import (
	"syscall"

	"github.com/appc/spec/schema"
	stage1commontypes "github.com/rkt/rkt/stage1/common/types"
	stage1initcommon "github.com/rkt/rkt/stage1/init/common"
)

type ProcessCredentials struct {
	uid               int
	gid               int
	supplementaryGIDs []int
}

func LookupProcessCredentials(pod *stage1commontypes.Pod, ra *schema.RuntimeApp, root string) (*ProcessCredentials, error) {
	var c ProcessCredentials
	var err error

	c.uid, c.gid, err = stage1initcommon.ParseUserGroup(pod, ra)
	if err != nil {
		return nil, err
	}

	// supplementary groups - ensure primary group is included
	n := len(ra.App.SupplementaryGIDs)
	c.supplementaryGIDs = make([]int, n+1, n+1)
	c.supplementaryGIDs[0] = c.gid
	copy(c.supplementaryGIDs[1:], ra.App.SupplementaryGIDs)

	return &c, nil
}

func SetProcessCredentials(c *ProcessCredentials) error {
	if err := syscall.Setresgid(c.gid, c.gid, c.gid); err != nil {
		return err
	}
	if err := syscall.Setgroups(c.supplementaryGIDs); err != nil {
		return err
	}
	if err := syscall.Setresuid(c.uid, c.uid, c.uid); err != nil {
		return err
	}

	return nil
}
