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

// +build host coreos src

package main

import (
	"fmt"
	"testing"

	"github.com/coreos/rkt/tests/testutils"
)

func TestMknod(t *testing.T) {
	ctx := testutils.NewRktRunCtx()
	defer ctx.Cleanup()

	image := getInspectImagePath()

	for _, tt := range []struct {
		rktParam string
		expect   string
	}{
		/* There should be no restriction on /dev/null */
		{
			rktParam: "--check-mknod=c:1:3:/null",
			expect:   "mknod /null: succeed",
		},

		/* Test the old ptmx device node, before devpts filesystem
		 * existed. It should be blocked. Containers should use the new
		 * ptmx from devpts instead. It is created with
		 * "mknod name c 5 2" according to:
		 * https://github.com/torvalds/linux/blob/master/Documentation/filesystems/devpts.txt
		 */
		{
			rktParam: "--check-mknod=c:5:2:/ptmx",
			expect:   "mknod /ptmx: fail",
		},

		/* /dev/loop-control has major:minor 10:237 according to:
		 * https://github.com/torvalds/linux/blob/master/Documentation/devices.txt#L424
		 */
		{
			rktParam: "--check-mknod=c:10:237:/loop-control",
			expect:   "mknod /loop-control: fail",
		},
	} {
		rktCmd := fmt.Sprintf(
			"%s --debug --insecure-options=image run %s --exec=/inspect -- %s",
			ctx.Cmd(), image, tt.rktParam)

		runRktAndCheckOutput(t, rktCmd, tt.expect, false)
	}
}
