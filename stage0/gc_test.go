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

package stage0

import (
	"strings"
	"testing"
)

var mountinfo = `8 7 0:1 / /my/mount/prefix/blah ignore this stuff
7 6 0:2 / /my/mount/prefix/foo bar
6 4 4.5 / /my/mount/prefix/foo car
9 7 0:3 / /my/mount/prefix/foo rar
2 6 0:5 / /my/mount/prefix/foo scar
5 0 0:18 / /not/my/mount/prefix sly dog`

func TestMountOrdering(t *testing.T) {
	tests := []struct {
		prefix     string
		ids        []int
		shouldPass bool
	}{
		{
			prefix:     "/my/mount/prefix",
			ids:        []int{2, 9, 8, 7, 6},
			shouldPass: true,
		},
	}

	for i, tt := range tests {
		mi := strings.NewReader(mountinfo)
		mnts, err := getMountsForPrefix(tt.prefix, mi)
		if err != nil {
			t.Errorf("problems finding mount points: %v", err)
		}

		if len(mnts) != len(tt.ids) {
			t.Errorf("test  %d: didn't find the expected number of mounts. found %d but wanted %d.", i, len(mnts), len(tt.ids))
			return
		}

		for j, mnt := range mnts {
			if mnt.id != tt.ids[j] {
				t.Errorf("test #%d: problem with mount ordering; mount at index %d is %d not %d",
					i, j, mnt.id, tt.ids[j])
				return
			}
		}
	}
}
