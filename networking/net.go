// Copyright 2015 CoreOS, Inc.
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

package networking

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"sort"

	"github.com/coreos/rocket/networking/util"
)

type Net struct {
	util.Net
	args string
}

const RktNetPath = "/etc/rkt-net.conf.d"

func listFiles(dir string) ([]string, error) {
	dirents, err := ioutil.ReadDir(RktNetPath)
	switch {
	case err == nil:
	case os.IsNotExist(err):
		return nil, nil
	default:
		return nil, err
	}

	files := []string{}
	for _, dent := range dirents {
		if dent.IsDir() {
			continue
		}

		files = append(files, dent.Name())
	}

	return files, nil
}

func LoadNets() ([]Net, error) {
	files, err := listFiles(RktNetPath)
	if err != nil {
		return nil, err
	}

	sort.Strings(files)

	nets := make([]Net, 0, len(files))

	for _, filename := range files {
		filepath := path.Join(RktNetPath, filename)
		n := Net{}
		if err := util.LoadNet(filepath, &n); err != nil {
			return nil, fmt.Errorf("error loading %v: %v", filepath, err)
		}

		nets = append(nets, n)
	}

	return nets, nil
}
