// Copyright 2015 The rkt Authors
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

package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"runtime"
	"syscall"

	"github.com/appc/spec/schema"
	"github.com/appc/spec/schema/types"
	"github.com/rkt/rkt/common"
	rktlog "github.com/rkt/rkt/pkg/log"
	stage1commontypes "github.com/rkt/rkt/stage1/common/types"
	stage1initcommon "github.com/rkt/rkt/stage1/init/common"
	"github.com/rkt/rkt/stage1_fly"
)

var (
	debug   bool
	podPid  string
	appName string

	log  *rktlog.Logger
	diag *rktlog.Logger
)

func init() {
	flag.BoolVar(&debug, "debug", false, "Run in debug mode")
	flag.StringVar(&podPid, "pid", "", "Pod PID")
	flag.StringVar(&appName, "appname", "", "Application name")

	log, diag, _ = rktlog.NewLogSet("fly-enter", false)
}

func getRootDir(pid string) (string, error) {
	rootLink := fmt.Sprintf("/proc/%s/root", pid)

	return os.Readlink(rootLink)
}

func getRuntimeApp(pm *schema.PodManifest) (*schema.RuntimeApp, error) {
	if len(pm.Apps) != 1 {
		return nil, fmt.Errorf("fly only supports 1 application per Pod for now")
	}

	return &pm.Apps[0], nil
}

func execArgs(envv []string) error {
	argv0 := flag.Arg(0)
	argv := flag.Args()

	return syscall.Exec(argv0, argv, envv)
}

func main() {
	flag.Parse()

	log.SetDebug(debug)
	diag.SetDebug(debug)

	if !debug {
		diag.SetOutput(ioutil.Discard)
	}

	// lock the current goroutine to its current OS thread.
	// This will force the subsequent syscalls *made by this goroutine only*
	// to be executed in the same OS thread as Setresuid, and Setresgid,
	// see https://github.com/golang/go/issues/1435#issuecomment-66054163.
	runtime.LockOSThread()

	root, err := getRootDir(podPid)
	if err != nil {
		log.FatalE("Failed to get pod root", err)
	}

	pm, err := stage1commontypes.LoadPodManifest(".")
	if err != nil {
		log.FatalE("Failed to load pod manifest", err)
	}

	ra, err := getRuntimeApp(pm)
	if err != nil {
		log.FatalE("Failed to get app", err)
	}

	// mock up a pod so we can call LookupProcessCredentials
	pod := &stage1commontypes.Pod{
		Root: root,
	}
	credentials, err := stage1_fly.LookupProcessCredentials(pod, ra, root)
	if err != nil {
		log.FatalE("failed to lookup process credentials", err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		log.FatalE("Failed to get cwd", err)
	}

	env, err := common.ReadEnvFileRaw(stage1initcommon.EnvFilePath(cwd, types.ACName(appName)))
	if err != nil {
		log.FatalE("Failed to read app env", err)
	}

	if err := os.Chdir(root); err != nil {
		log.FatalE("Failed to change to new root", err)
	}

	if err := syscall.Chroot(root); err != nil {
		log.FatalE("Failed to chroot", err)
	}

	diag.Printf("setting credentials: %+v", credentials)
	if err := stage1_fly.SetProcessCredentials(credentials); err != nil {
		log.FatalE("can't set process credentials", err)
	}

	diag.Println("PID:", podPid)
	diag.Println("APP:", appName)
	diag.Println("ARGS:", flag.Args())

	if err := execArgs(env); err != nil {
		log.PrintE("exec failed", err)
	}

	os.Exit(254)
}
