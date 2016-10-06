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
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/coreos/rkt/common"
	rktlog "github.com/coreos/rkt/pkg/log"
	stage1initcommon "github.com/coreos/rkt/stage1/init/common"

	"github.com/appc/spec/schema/types"
)

var (
	debug bool
	log   *rktlog.Logger
	diag  *rktlog.Logger
)

func init() {
	flag.BoolVar(&debug, "debug", false, "Run in debug mode")
}

// TODO use named flags instead of positional
func main() {
	flag.Parse()

	stage1initcommon.InitDebug(debug)

	log, diag, _ = rktlog.NewLogSet("stage1", debug)
	if !debug {
		diag.SetOutput(ioutil.Discard)
	}

	appName, err := types.NewACName(flag.Arg(1))
	if err != nil {
		log.PrintE("invalid app name", err)
		os.Exit(254)
	}

	enterEP := flag.Arg(2)

	args := []string{enterEP}

	args = append(args, fmt.Sprintf("--pid=%s", flag.Arg(3)))
	args = append(args, "/usr/bin/systemctl")
	args = append(args, "is-active")
	args = append(args, appName.String())

	cmd := exec.Cmd{
		Path: args[0],
		Args: args,
	}

	// rely only on the output, since it returns non-zero for inactive units
	out, _ := cmd.Output()

	if string(out) != "inactive\n" {
		log.Printf("app %q is still running", appName.String())
		os.Exit(254)
	}

	s1rootfs := common.Stage1RootfsPath(".")
	serviceDir := filepath.Join(s1rootfs, "usr", "lib", "systemd", "system")
	appServicePaths := []string{
		filepath.Join(serviceDir, appName.String()+".service"),
		filepath.Join(serviceDir, "reaper-"+appName.String()+".service"),
	}

	for _, p := range appServicePaths {
		if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
			log.PrintE("error removing app service file", err)
			os.Exit(254)
		}
	}

	args = []string{enterEP}
	args = append(args, fmt.Sprintf("--pid=%s", flag.Arg(3)))
	args = append(args, "/usr/bin/systemctl")
	args = append(args, "daemon-reload")

	cmd = exec.Cmd{
		Path: args[0],
		Args: args,
	}
	if err := cmd.Run(); err != nil {
		log.PrintE(`error executing "systemctl daemon-reload"`, err)
		os.Exit(254)
	}

	// TODO unmount all the volumes

	os.Exit(0)
}
