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

	"github.com/coreos/go-systemd/unit"
	rktlog "github.com/coreos/rkt/pkg/log"
	stage1common "github.com/coreos/rkt/stage1/common"
	stage1types "github.com/coreos/rkt/stage1/common/types"
	stage1initcommon "github.com/coreos/rkt/stage1/init/common"

	"github.com/appc/spec/schema/types"
)

var (
	flagApp  string
	flagUUID string
	debug    bool
	log      *rktlog.Logger
	diag     *rktlog.Logger
)

func init() {
	flag.StringVar(&flagApp, "app", "", "Application name")
	flag.StringVar(&flagUUID, "uuid", "", "Pod UUID")
	flag.BoolVar(&debug, "debug", false, "Run in debug mode")
}

func main() {
	flag.Parse()

	stage1initcommon.InitDebug(debug)

	log, diag, _ = rktlog.NewLogSet("app-add", debug)
	if !debug {
		diag.SetOutput(ioutil.Discard)
	}

	uuid, err := types.NewUUID(flagUUID)
	if err != nil {
		log.FatalE("UUID is missing or malformed", err)
	}

	appName, err := types.NewACName(flagApp)
	if err != nil {
		log.FatalE("invalid app name", err)
	}

	root := "."
	p, err := stage1types.LoadPod(root, uuid, nil)
	if err != nil {
		log.FatalE("failed to load pod", err)
	}

	ra := p.Manifest.Apps.Get(*appName)
	if ra == nil {
		log.Fatalf("failed to find app %q", *appName)
	}

	binPath, err := stage1initcommon.FindBinPath(p, ra)
	if err != nil {
		log.FatalE("failed to find bin path", err)
	}

	if ra.App.WorkingDirectory == "" {
		ra.App.WorkingDirectory = "/"
	}

	enterCmd := stage1common.PrepareEnterCmd(false)
	err = stage1initcommon.AppAddMounts(p, ra, enterCmd)
	if err != nil {
		log.FatalE("error adding app mounts", err)
	}

	// write service files
	w := stage1initcommon.NewUnitWriter(p)
	w.AppUnit(ra, binPath,
		unit.NewUnitOption("Unit", "Before", "halt.target"),
		unit.NewUnitOption("Unit", "Conflicts", "halt.target"),
		unit.NewUnitOption("Service", "StandardOutput", "journal+console"),
		unit.NewUnitOption("Service", "StandardError", "journal+console"),
	)
	w.AppReaperUnit(ra.Name, binPath)
	if err := w.Error(); err != nil {
		log.FatalE("error generating app units", err)
	}

	// stage2 environment is ready at this point, but systemd does not know
	// about the new application yet
	args := enterCmd
	args = append(args, "/usr/bin/systemctl")
	args = append(args, "daemon-reload")

	cmd := exec.Cmd{
		Path: args[0],
		Args: args,
	}

	if out, err := cmd.CombinedOutput(); err != nil {
		log.Fatalf("%q failed at daemon-reload:\n%s", appName, out)
	}

	os.Exit(0)
}
