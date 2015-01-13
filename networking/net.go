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
	"io/ioutil"
	"log"
	"os"
	"path"

	"github.com/coreos/rocket/networking/util"
)

type Net struct {
	util.Net
	args string
}

const RktNetPath = "/etc/rkt-net.conf.d"
const DefaultIPNet = "172.16.28.0/24"

var defaultNet = Net{
	Net: util.Net{
		Name: "default",
		Type: "veth",
	},
	args: "default,iprange=" + DefaultIPNet,
}

func LoadNets() ([]Net, error) {
	dirents, err := ioutil.ReadDir(RktNetPath)
	switch {
	case err == nil:
	case os.IsNotExist(err):
		return nil, nil
	default:
		return nil, err
	}

	var nets []Net

	for _, dent := range dirents {
		if dent.IsDir() {
			continue
		}

		nf := path.Join(RktNetPath, dent.Name())
		n := Net{}
		if err := util.LoadNet(nf, &n); err != nil {
			log.Printf("Error loading %v: %v", nf, err)
			continue
		}

		nets = append(nets, n)
	}

	nets = append(nets, defaultNet)

	return nets, nil }
