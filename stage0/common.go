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

package stage0

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/coreos/rkt/common"
	"github.com/hashicorp/errwrap"
)

// RunCrossingEntrypoint wraps the execution of a stage1 entrypoint which
// requires crossing the stage0/stage1/stage2 boundary during its execution,
// by setting up proper environment variables for enter.
func RunCrossingEntrypoint(dir string, podPID int, appName string, entrypoint string, entrypointArgs []string) error {
	enterCmd, err := getStage1Entrypoint(dir, enterEntrypoint)
	if err != nil {
		return errwrap.Wrap(errors.New("error determining 'enter' entrypoint"), err)
	}

	previousDir, err := os.Getwd()
	if err != nil {
		return err
	}

	if err := os.Chdir(dir); err != nil {
		return errwrap.Wrap(errors.New("failed changing to dir"), err)
	}

	ep, err := getStage1Entrypoint(dir, entrypoint)
	if err != nil {
		return fmt.Errorf("%q not implemented for pod's stage1: %v", entrypoint, err)
	}
	execArgs := []string{filepath.Join(common.Stage1RootfsPath(dir), ep)}
	execArgs = append(execArgs, entrypointArgs...)

	c := exec.Cmd{
		Path:   execArgs[0],
		Args:   execArgs,
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Env: []string{
			fmt.Sprintf("%s=%s", common.CrossingEnterCmd, filepath.Join(common.Stage1RootfsPath(dir), enterCmd)),
			fmt.Sprintf("%s=%d", common.CrossingEnterPID, podPID),
		},
	}

	if err := c.Run(); err != nil {
		return fmt.Errorf("error executing stage1 entrypoint: %v", err)
	}

	if err := os.Chdir(previousDir); err != nil {
		return errwrap.Wrap(errors.New("failed changing to dir"), err)
	}

	return nil
}
