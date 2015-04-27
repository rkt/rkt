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

package plugin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/coreos/rkt/Godeps/_workspace/src/github.com/appc/cni/pkg/ip"
	"github.com/coreos/rkt/Godeps/_workspace/src/github.com/vishvananda/netlink" // Find returns the full path of the plugin by searching in CNI_PATH
)

func Find(plugin string) string {
	paths := strings.Split(os.Getenv("CNI_PATH"), ":")

	for _, p := range paths {
		fullname := filepath.Join(p, plugin)
		if fi, err := os.Stat(fullname); err == nil && fi.Mode().IsRegular() {
			return fullname
		}
	}

	return ""
}

// ExecAdd executes IPAM plugin, assuming CNI_COMMAND == ADD.
// Parses and returns resulting IPConfig
func ExecAdd(plugin string, netconf []byte) (*Result, error) {
	if os.Getenv("CNI_COMMAND") != "ADD" {
		return nil, fmt.Errorf("CNI_COMMAND is not ADD")
	}

	pluginPath := Find(plugin)
	if pluginPath == "" {
		return nil, fmt.Errorf("could not find %q plugin", plugin)
	}

	stdout := &bytes.Buffer{}

	c := exec.Cmd{
		Path:   pluginPath,
		Args:   []string{pluginPath},
		Stdin:  bytes.NewBuffer(netconf),
		Stdout: stdout,
		Stderr: os.Stderr,
	}
	if err := c.Run(); err != nil {
		return nil, err
	}

	res := &Result{}
	err := json.Unmarshal(stdout.Bytes(), res)
	return res, err
}

// ExecDel executes IPAM plugin, assuming CNI_COMMAND == DEL.
func ExecDel(plugin string, netconf []byte) error {
	if os.Getenv("CNI_COMMAND") != "DEL" {
		return fmt.Errorf("CNI_COMMAND is not DEL")
	}

	pluginPath := Find(plugin)
	if pluginPath == "" {
		return fmt.Errorf("could not find %q plugin", plugin)
	}

	c := exec.Cmd{
		Path:   pluginPath,
		Args:   []string{pluginPath},
		Stdin:  bytes.NewBuffer(netconf),
		Stderr: os.Stderr,
	}
	return c.Run()
}

// ConfigureIface takes the result of IPAM plugin and
// applies to the ifName interface
func ConfigureIface(ifName string, res *Result) error {
	link, err := netlink.LinkByName(ifName)
	if err != nil {
		return fmt.Errorf("failed to lookup %q: %v", ifName, err)
	}

	if err := netlink.LinkSetUp(link); err != nil {
		return fmt.Errorf("failed too set %q UP: %v", ifName, err)
	}

	// TODO(eyakubovich): IPv6
	addr := &netlink.Addr{IPNet: &res.IP4.IP, Label: ""}
	if err = netlink.AddrAdd(link, addr); err != nil {
		return fmt.Errorf("failed to add IP addr to %q: %v", ifName, err)
	}

	for _, r := range res.IP4.Routes {
		gw := r.GW
		if gw == nil {
			gw = res.IP4.Gateway
		}
		if err = ip.AddRoute(&r.Dst, gw, link); err != nil {
			// we skip over duplicate routes as we assume the first one wins
			if !os.IsExist(err) {
				return fmt.Errorf("failed to add route '%v via %v dev %v': %v", r.Dst, gw, ifName, err)
			}
		}
	}

	return nil
}

// PrintResult writes out prettified Result to stdout
func PrintResult(res *Result) error {
	data, err := json.MarshalIndent(res, "", "    ")
	if err != nil {
		return err
	}
	_, err = os.Stdout.Write(data)
	return err
}
