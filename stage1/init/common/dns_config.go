// Copyright 2016 The rkt Authors
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

package common

import (
	"path/filepath"

	_common "github.com/coreos/rkt/common"
)

/*
 Bind-mount the hosts /etc/resolv.conf in to the stage1's /etc/rkt-resolv.conf.
 That file will then be bind-mounted in to the stage2 by perpare-app.c
*/
func UseHostResolv(podRoot string) error {
	return BindMount(
		"/etc/resolv.conf",
		filepath.Join(_common.Stage1RootfsPath(podRoot), "etc/rkt-resolv.conf"),
		true)
}

/*
 Bind-mount the hosts /etc/hosts in to the stage1's /etc/rkt-hosts
 That file will then be bind-mounted in to the stage2 by perpare-app.c
*/
func UseHostHosts(podRoot string) error {
	return BindMount(
		"/etc/hosts",
		filepath.Join(_common.Stage1RootfsPath(podRoot), "etc/rkt-hosts"),
		true)
}
