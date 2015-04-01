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
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/coreos/rkt/common"
	rktnet "github.com/coreos/rkt/networking/net"
)

// TODO(eyakubovich): make this configurable in rkt.conf
const UserNetPluginsPath = "/usr/lib/rkt/plugins/net"
const BuiltinNetPluginsPath = "usr/lib/rkt/plugins/net"

func (e *podEnv) netPluginAdd(n *Net, netns, args, ifName string) (ip, hostIP net.IP, err error) {
	output, err := e.execNetPlugin("ADD", n, netns, args, ifName)
	if err != nil {
		return nil, nil, err
	}

	ifConf := rktnet.IfConfig{}
	if err = json.Unmarshal(output, &ifConf); err != nil {
		return nil, nil, fmt.Errorf("error parsing %q output: %v", n.Name, err)
	}

	return ifConf.IP, ifConf.HostIP, nil
}

func (e *podEnv) netPluginDel(n *Net, netns, args, ifName string) error {
	_, err := e.execNetPlugin("DEL", n, netns, args, ifName)
	return err
}

func (e *podEnv) pluginPaths() []string {
	// try 3rd-party path first
	return []string{
		UserNetPluginsPath,
		filepath.Join(common.Stage1RootfsPath(e.rktRoot), BuiltinNetPluginsPath),
	}
}

func (e *podEnv) findNetPlugin(plugin string) string {
	for _, p := range e.pluginPaths() {
		fullname := filepath.Join(p, plugin)
		if fi, err := os.Stat(fullname); err == nil && fi.Mode().IsRegular() {
			return fullname
		}
	}

	return ""
}

func envVars(vars [][2]string) []string {
	env := os.Environ()

	for _, kv := range vars {
		env = append(env, strings.Join(kv[:], "="))
	}

	return env
}

func (e *podEnv) execNetPlugin(cmd string, n *Net, netns, args, ifName string) ([]byte, error) {
	pluginPath := e.findNetPlugin(n.Type)
	if pluginPath == "" {
		return nil, fmt.Errorf("Could not find plugin %q", n.Type)
	}

	vars := [][2]string{
		{"RKT_NETPLUGIN_COMMAND", cmd},
		{"RKT_NETPLUGIN_PODID", e.podID.String()},
		{"RKT_NETPLUGIN_NETNS", netns},
		{"RKT_NETPLUGIN_ARGS", args},
		{"RKT_NETPLUGIN_IFNAME", ifName},
		{"RKT_NETPLUGIN_NETNAME", n.Name},
		{"RKT_NETPLUGIN_NETCONF", n.Filename},
		{"RKT_NETPLUGIN_IPAMPATH", strings.Join(e.pluginPaths(), ":")},
	}

	stdout := &bytes.Buffer{}

	c := exec.Cmd{
		Path:   pluginPath,
		Args:   []string{pluginPath},
		Env:    envVars(vars),
		Stdout: stdout,
		Stderr: os.Stderr,
	}

	if err := c.Run(); err != nil {
		return nil, err
	}

	return stdout.Bytes(), nil
}
