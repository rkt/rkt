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
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"syscall"

	"github.com/appc/spec/schema"
	"github.com/hashicorp/errwrap"
	rktlog "github.com/rkt/rkt/pkg/log"
	"github.com/rkt/rkt/stage1_fly"
)

const (
	flavor = "fly"
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
	flag.StringVar(&appName, "appname", "", "Application (Ignored in rkt fly)")

	log, diag, _ = rktlog.NewLogSet("fly-enter", false)
}

func getRootDir(pid string) (string, error) {
	rootLink := fmt.Sprintf("/proc/%s/root", pid)

	return os.Readlink(rootLink)
}

func getPodManifest() (*schema.PodManifest, error) {
	f, err := os.Open("pod")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	pmb, err := ioutil.ReadAll(f)

	if err != nil {
		return nil, errwrap.Wrap(errors.New("error reading pod manifest"), err)
	}
	pm := &schema.PodManifest{}
	if err = pm.UnmarshalJSON(pmb); err != nil {
		return nil, errwrap.Wrap(errors.New("invalid pod manifest"), err)
	}
	return pm, nil
}

func getRuntimeApp(pm *schema.PodManifest) (*schema.RuntimeApp, error) {
	if len(pm.Apps) != 1 {
		return nil, fmt.Errorf("flavor %q only supports 1 application per Pod for now", flavor)
	}

	return &pm.Apps[0], nil
}

func execArgs() error {
	argv0 := flag.Arg(0)
	argv := flag.Args()
	envv := []string{}

	return syscall.Exec(argv0, argv, envv)
}

func main() {
	flag.Parse()

	log.SetDebug(debug)
	diag.SetDebug(debug)

	if !debug {
		diag.SetOutput(ioutil.Discard)
	}

	root, err := getRootDir(podPid)
	if err != nil {
		log.FatalE("Failed to get pod root", err)
	}

	pm, err := getPodManifest()
	if err != nil {
		log.FatalE("Failed to get pod manifest", err)
	}

	ra, err := getRuntimeApp(pm)
	if err != nil {
		log.FatalE("Failed to get app", err)
	}

	credentials, err := stage1_fly.LookupProcessCredentials(ra, root)
	if err != nil {
		log.FatalE("failed to lookup process credentials", err)
	}

	if err := os.Chdir(root); err != nil {
		log.FatalE("Failed to change to new root", err)
	}

	if err := syscall.Chroot(root); err != nil {
		log.FatalE("Failed to chroot", err)
	}

	if err := stage1_fly.SetProcessCredentials(credentials, diag); err != nil {
		log.FatalE("can't set process credentials", err)
	}

	diag.Println("PID:", podPid)
	diag.Println("APP:", appName)
	diag.Println("ARGS:", flag.Args())

	if err := execArgs(); err != nil {
		log.PrintE("exec failed", err)
	}

	os.Exit(254)
}
