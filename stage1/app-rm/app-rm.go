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

//+build linux

package main

import (
	"flag"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/coreos/rkt/common"
	rktlog "github.com/coreos/rkt/pkg/log"
	stage1common "github.com/coreos/rkt/stage1/common"
	stage1initcommon "github.com/coreos/rkt/stage1/init/common"

	"github.com/appc/spec/schema/types"
)

var (
	flagApp string
	debug   bool
	log     *rktlog.Logger
	diag    *rktlog.Logger
)

func init() {
	flag.StringVar(&flagApp, "app", "", "Application name")
	flag.BoolVar(&debug, "debug", false, "Run in debug mode")
}

func main() {
	flag.Parse()

	stage1initcommon.InitDebug(debug)

	log, diag, _ = rktlog.NewLogSet("app-rm", debug)
	if !debug {
		diag.SetOutput(ioutil.Discard)
	}

	appName, err := types.NewACName(flagApp)
	if err != nil {
		log.FatalE("invalid app name", err)
	}

	enterCmd := stage1common.PrepareEnterCmd(false)

	args := enterCmd
	args = append(args, "/usr/bin/systemctl")
	args = append(args, "is-active")
	args = append(args, appName.String())

	cmd := exec.Cmd{
		Path: args[0],
		Args: args,
	}

	// rely only on the output, since is-active returns non-zero for inactive units
	out, _ := cmd.Output()

	if string(out) != "inactive\n" {
		log.Fatalf("app %q is still running", appName.String())
	}

	s1rootfs := common.Stage1RootfsPath(".")
	serviceDir := filepath.Join(s1rootfs, "usr", "lib", "systemd", "system")
	appServicePaths := []string{
		filepath.Join(serviceDir, appName.String()+".service"),
		filepath.Join(serviceDir, "reaper-"+appName.String()+".service"),
	}

	for _, p := range appServicePaths {
		if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
			log.FatalE("error removing app service file", err)
		}
	}

	args = enterCmd
	args = append(args, "/usr/bin/systemctl")
	args = append(args, "daemon-reload")

	cmd = exec.Cmd{
		Path: args[0],
		Args: args,
	}

	if out, err := cmd.CombinedOutput(); err != nil {
		log.Fatalf("%q failed at daemon-reload:\n%s", appName, out)
	}

	// TODO unmount all the volumes

	os.Exit(0)
}
