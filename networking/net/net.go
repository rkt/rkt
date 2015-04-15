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

package net

import (
	"encoding/json"
	"io/ioutil"
	gonet "net"
	"os"
)

// Net describes a network.
type Net struct {
	Name string `json:"name,omitempty"`
	Type string `json:"type,omitempty"`
	IPAM struct {
		Type string `json:"type,omitempty"`
	} `json:"ipam,omitempty"`
}

// LoadNet loads a JSON-encoded Net from the filesystem.
func LoadNet(path string, n interface{}) error {
	c, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	return json.Unmarshal(c, n)
}

// this is what net plugin returns to rkt
type IfConfig struct {
	IP  gonet.IP `json:"ip,omitempty"`
	IP6 gonet.IP `json:"ip6,omitempty"`

	// these are "extensions" and only meaningful for default net
	HostIP  gonet.IP `json:"hostIP,omitempty"`
	HostIP6 gonet.IP `json:"hostIP6,omitempty"`
}

func PrintIfConfig(conf *IfConfig) error {
	data, err := json.MarshalIndent(conf, "", "    ")
	if err != nil {
		return err
	}
	_, err = os.Stdout.Write(data)
	return err
}
