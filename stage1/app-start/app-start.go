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

	rktlog "github.com/coreos/rkt/pkg/log"
	stage1types "github.com/coreos/rkt/stage1/common/types"
	stage1initcommon "github.com/coreos/rkt/stage1/init/common"

	"github.com/appc/spec/schema/types"
	"github.com/coreos/go-systemd/unit"
)

var (
	debug               bool
	disableCapabilities bool
	disablePaths        bool
	disableSeccomp      bool
	privateUsers        string
	log                 *rktlog.Logger
	diag                *rktlog.Logger
)

func init() {
	flag.BoolVar(&debug, "debug", false, "Run in debug mode")
	flag.BoolVar(&disableCapabilities, "disable-capabilities-restriction", false, "Disable capability restrictions")
	flag.BoolVar(&disablePaths, "disable-paths", false, "Disable paths restrictions")
	flag.BoolVar(&disableSeccomp, "disable-seccomp", false, "Disable seccomp restrictions")
	flag.StringVar(&privateUsers, "private-users", "", "Run within user namespace. Can be set to [=UIDBASE[:NUIDS]]")
}

// TODO use named flags instead of positional
func main() {
	flag.Parse()

	stage1initcommon.InitDebug(debug)

	log, diag, _ = rktlog.NewLogSet("stage1", debug)
	if !debug {
		diag.SetOutput(ioutil.Discard)
	}

	uuid, err := types.NewUUID(flag.Arg(0))
	if err != nil {
		log.PrintE("UUID is missing or malformed", err)
		os.Exit(1)
	}

	appName, err := types.NewACName(flag.Arg(1))
	if err != nil {
		log.PrintE("invalid app name", err)
		os.Exit(1)
	}

	enterEP := flag.Arg(2)

	root := "."
	p, err := stage1types.LoadPod(root, uuid)
	if err != nil {
		log.PrintE("failed to load pod", err)
		os.Exit(1)
	}

	insecureOptions := stage1initcommon.Stage1InsecureOptions{
		DisablePaths:        disablePaths,
		DisableCapabilities: disableCapabilities,
		DisableSeccomp:      disableSeccomp,
	}

	ra := p.Manifest.Apps.Get(*appName)
	if ra == nil {
		log.Printf("failed to get app")
		os.Exit(1)
	}

	if ra.App.WorkingDirectory == "" {
		ra.App.WorkingDirectory = "/"
	}

	binPath, err := stage1initcommon.FindBinPath(p, ra)
	if err != nil {
		log.PrintE("failed to find bin path", err)
		os.Exit(1)
	}

	w := stage1initcommon.NewUnitWriter(p)

	w.AppUnit(ra, binPath, privateUsers, insecureOptions,
		unit.NewUnitOption("Unit", "Before", "halt.target"),
		unit.NewUnitOption("Unit", "Conflicts", "halt.target"),
		unit.NewUnitOption("Service", "StandardOutput", "journal+console"),
		unit.NewUnitOption("Service", "StandardError", "journal+console"),
	)

	w.AppReaperUnit(ra.Name, binPath)

	if err := w.Error(); err != nil {
		log.PrintE("error generating app units", err)
		os.Exit(1)
	}

	args := []string{enterEP}

	args = append(args, fmt.Sprintf("--pid=%s", flag.Arg(3)))
	args = append(args, "/usr/bin/systemctl")
	args = append(args, "daemon-reload")

	cmd := exec.Cmd{
		Path: args[0],
		Args: args,
	}

	if err := cmd.Run(); err != nil {
		log.PrintE("error executing daemon-reload", err)
		os.Exit(1)
	}

	args = []string{enterEP}

	args = append(args, fmt.Sprintf("--pid=%s", flag.Arg(3)))
	args = append(args, "/usr/bin/systemctl")
	args = append(args, "start")
	args = append(args, appName.String())

	cmd = exec.Cmd{
		Path: args[0],
		Args: args,
	}

	if err := cmd.Run(); err != nil {
		log.PrintE(fmt.Sprintf("error starting app %q", appName.String()), err)
		os.Exit(1)
	}

	// TODO unmount all the volumes

	os.Exit(0)
}
