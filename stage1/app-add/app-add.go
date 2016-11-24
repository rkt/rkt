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
	"errors"
	"flag"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/coreos/go-systemd/unit"
	"github.com/coreos/rkt/common/cgroup"
	"github.com/coreos/rkt/common/cgroup/v1"
	rktlog "github.com/coreos/rkt/pkg/log"
	stage1common "github.com/coreos/rkt/stage1/common"
	stage1types "github.com/coreos/rkt/stage1/common/types"
	stage1initcommon "github.com/coreos/rkt/stage1/init/common"
	"github.com/hashicorp/errwrap"

	"github.com/appc/spec/schema"
	"github.com/appc/spec/schema/types"
)

var (
	flagApp             string
	flagUUID            string
	debug               bool
	disableCapabilities bool
	disablePaths        bool
	disableSeccomp      bool
	privateUsers        string
	log                 *rktlog.Logger
	diag                *rktlog.Logger
)

func init() {
	flag.StringVar(&flagApp, "app", "", "Application name")
	flag.StringVar(&flagUUID, "uuid", "", "Pod UUID")
	flag.BoolVar(&debug, "debug", false, "Run in debug mode")
	flag.BoolVar(&disableCapabilities, "disable-capabilities-restriction", false, "Disable capability restrictions")
	flag.BoolVar(&disablePaths, "disable-paths", false, "Disable paths restrictions")
	flag.BoolVar(&disableSeccomp, "disable-seccomp", false, "Disable seccomp restrictions")
	flag.StringVar(&privateUsers, "private-users", "", "Run within user namespace. Can be set to [=UIDBASE[:NUIDS]]")
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
	p, err := stage1types.LoadPod(root, uuid)
	if err != nil {
		log.FatalE("failed to load pod", err)
	}

	flavor, _, err := stage1initcommon.GetFlavor(p)
	if err != nil {
		log.FatalE("failed to get stage1 flavor", err)
	}

	insecureOptions := stage1initcommon.Stage1InsecureOptions{
		DisablePaths:        disablePaths,
		DisableCapabilities: disableCapabilities,
		DisableSeccomp:      disableSeccomp,
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
	stage1initcommon.AppAddMounts(p, ra, enterCmd)

	// when using host cgroups, make the subgroup writable by pod systemd
	if flavor != "kvm" {
		err = prepareAppCgroups(p, ra, enterCmd)
		if err != nil {
			log.FatalE("error preparing cgroups", err)
		}
	}

	// write service files
	w := stage1initcommon.NewUnitWriter(p)
	w.AppUnit(ra, binPath, privateUsers, insecureOptions,
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

// prepareAppCgroups makes the cgroups-v1 hierarchy for this application writable
// by pod supervisor
func prepareAppCgroups(p *stage1types.Pod, ra *schema.RuntimeApp, enterCmd []string) error {
	isUnified, err := cgroup.IsCgroupUnified("/")
	if err != nil {
		return errwrap.Wrap(errors.New("failed to determine cgroup version"), err)
	}

	if isUnified {
		return nil
	}

	enabledCgroups, err := v1.GetEnabledCgroups()
	if err != nil {
		return errwrap.Wrap(errors.New("error getting cgroups"), err)
	}

	b, err := ioutil.ReadFile(filepath.Join(p.Root, "subcgroup"))
	if err != nil {
		log.PrintE("continuing with per-app isolators disabled", err)
		return nil
	}

	subcgroup := string(b)
	serviceName := stage1initcommon.ServiceUnitName(ra.Name)
	if err := v1.RemountCgroupKnobsRW(enabledCgroups, subcgroup, serviceName, enterCmd); err != nil {
		return errwrap.Wrap(errors.New("error restricting application cgroups"), err)
	}

	return nil
}
