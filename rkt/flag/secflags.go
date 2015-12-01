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

package flag

const (
	insecureNone  = 0
	insecureImage = 1 << (iota - 1)
	insecureTls
	insecureOnDisk

	insecureAll = (insecureImage | insecureTls | insecureOnDisk)
)

var (
	InsecureOptions = []string{"none", "image", "tls", "ondisk", "all"}

	InsecureOptionsMap = map[string]int{
		InsecureOptions[0]: insecureNone,
		InsecureOptions[1]: insecureImage,
		InsecureOptions[2]: insecureTls,
		InsecureOptions[3]: insecureOnDisk,
		InsecureOptions[4]: insecureAll,
	}
)

type SecFlags BitFlags

func (sf *SecFlags) SkipImageCheck() bool {
	return (*BitFlags)(sf).hasFlag(insecureImage)
}

func (sf *SecFlags) SkipTlsCheck() bool {
	return (*BitFlags)(sf).hasFlag(insecureTls)
}

func (sf *SecFlags) SkipOnDiskCheck() bool {
	return (*BitFlags)(sf).hasFlag(insecureOnDisk)
}

func (sf *SecFlags) SkipAllSecurityChecks() bool {
	return (*BitFlags)(sf).hasFlag(insecureAll)
}

func (sf *SecFlags) SkipAnySecurityChecks() bool {
	return (*BitFlags)(sf).flags != 0
}
