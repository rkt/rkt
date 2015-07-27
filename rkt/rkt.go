// Copyright 2014 The rkt Authors
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
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/coreos/rkt/Godeps/_workspace/src/github.com/spf13/cobra"
	"github.com/coreos/rkt/common"
	"github.com/coreos/rkt/pkg/keystore"
	"github.com/coreos/rkt/pkg/multicall"
	"github.com/coreos/rkt/rkt/config"
	rktflag "github.com/coreos/rkt/rkt/flag"
)

const (
	cliName        = "rkt"
	cliDescription = "rkt, the application container runner"

	defaultDataDir = "/var/lib/rkt"
)

type absDir string

func (d *absDir) String() string {
	return (string)(*d)
}

func (d *absDir) Set(str string) error {
	if str == "" {
		return fmt.Errorf(`"" is not a valid directory`)
	}

	dir, err := filepath.Abs(str)
	if err != nil {
		return err
	}
	*d = (absDir)(dir)
	return nil
}

func (d *absDir) Type() string {
	return "absolute-directory"
}

var (
	tabOut      *tabwriter.Writer
	globalFlags = struct {
		Dir                string
		SystemConfigDir    string
		LocalConfigDir     string
		Debug              bool
		Help               bool
		InsecureFlags      *rktflag.SecFlags
		TrustKeysFromHttps bool
	}{
		Dir:             defaultDataDir,
		SystemConfigDir: common.DefaultSystemConfigDir,
		LocalConfigDir:  common.DefaultLocalConfigDir,
	}

	cachedConfig  *config.Config
	cachedDataDir string
	cmdExitCode   int
)

var cmdRkt = &cobra.Command{
	Use:   "rkt [command]",
	Short: cliDescription,
}

func init() {
	sf, err := rktflag.NewSecFlags("none")
	if err != nil {
		stderr("rkt: problem initializing: %v", err)
		os.Exit(1)
	}

	globalFlags.InsecureFlags = sf

	cmdRkt.PersistentFlags().BoolVar(&globalFlags.Debug, "debug", false, "print out more debug information to stderr")
	cmdRkt.PersistentFlags().Var((*absDir)(&globalFlags.Dir), "dir", "rkt data directory")
	cmdRkt.PersistentFlags().Var((*absDir)(&globalFlags.SystemConfigDir), "system-config", "system configuration directory")
	cmdRkt.PersistentFlags().Var((*absDir)(&globalFlags.LocalConfigDir), "local-config", "local configuration directory")
	cmdRkt.PersistentFlags().Var(globalFlags.InsecureFlags, "insecure-options",
		fmt.Sprintf("comma-separated list of security features to disable. Allowed values: %s",
			globalFlags.InsecureFlags.PermissibleString()))
	cmdRkt.PersistentFlags().BoolVar(&globalFlags.TrustKeysFromHttps, "trust-keys-from-https",
		true, "automatically trust gpg keys fetched from https")

	// TODO: Remove before 1.0
	rktflag.InstallDeprecatedSkipVerify(cmdRkt.PersistentFlags(), sf)

	cobra.EnablePrefixMatching = true
}

func getTabOutWithWriter(writer io.Writer) *tabwriter.Writer {
	aTabOut := new(tabwriter.Writer)
	aTabOut.Init(writer, 0, 8, 1, '\t', 0)
	return aTabOut
}

// runWrapper return a func(cmd *cobra.Command, args []string) that internally
// will add command function return code and the reinsertion of the "--" flag
// terminator.
func runWrapper(cf func(cmd *cobra.Command, args []string) (exit int)) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		cmdExitCode = cf(cmd, args)
	}
}

func main() {
	// check if rkt is executed with a multicall command
	multicall.MaybeExec()

	cmdRkt.SetUsageFunc(usageFunc)

	// Make help just show the usage
	cmdRkt.SetHelpTemplate(`{{.UsageString}}`)

	cmdRkt.Execute()
	os.Exit(cmdExitCode)
}

func stderr(format string, a ...interface{}) {
	out := fmt.Sprintf(format, a...)
	fmt.Fprintln(os.Stderr, strings.TrimSuffix(out, "\n"))
}

func stdout(format string, a ...interface{}) {
	out := fmt.Sprintf(format, a...)
	fmt.Fprintln(os.Stdout, strings.TrimSuffix(out, "\n"))
}

// where pod directories are created and locked before moving to prepared
func embryoDir() string {
	return filepath.Join(getDataDir(), "pods", "embryo")
}

// where pod trees reside during (locked) and after failing to complete preparation (unlocked)
func prepareDir() string {
	return filepath.Join(getDataDir(), "pods", "prepare")
}

// where pod trees reside upon successful preparation
func preparedDir() string {
	return filepath.Join(getDataDir(), "pods", "prepared")
}

// where pod trees reside once run
func runDir() string {
	return filepath.Join(getDataDir(), "pods", "run")
}

// where pod trees reside once exited & marked as garbage by a gc pass
func exitedGarbageDir() string {
	return filepath.Join(getDataDir(), "pods", "exited-garbage")
}

// where never-executed pod trees reside once marked as garbage by a gc pass (failed prepares, expired prepareds)
func garbageDir() string {
	return filepath.Join(getDataDir(), "pods", "garbage")
}

func getKeystore() *keystore.Keystore {
	if globalFlags.InsecureFlags.SkipImageCheck() {
		return nil
	}
	config := keystore.NewConfig(globalFlags.SystemConfigDir, globalFlags.LocalConfigDir)
	return keystore.New(config)
}

func getDataDir() string {
	if cachedDataDir == "" {
		cachedDataDir = calculateDataDir()
	}
	return cachedDataDir
}

func calculateDataDir() string {
	// If --dir parameter is passed, then use this value.
	if dirFlag := cmdRkt.PersistentFlags().Lookup("dir"); dirFlag != nil {
		if dirFlag.Changed {
			return globalFlags.Dir
		}
	} else {
		// should not happen
		panic(`"--dir" flag not found`)
	}

	// If above fails, then try to get the value from configuration.
	if config, err := getConfig(); err != nil {
		stderr("rkt: cannot get configuration: %v", err)
		os.Exit(1)
	} else {
		if config.Paths.DataDir != "" {
			return config.Paths.DataDir
		}
	}

	// If above fails, then use the default.
	return defaultDataDir
}

func getConfig() (*config.Config, error) {
	if cachedConfig == nil {
		cfg, err := config.GetConfigFrom(globalFlags.SystemConfigDir, globalFlags.LocalConfigDir)
		if err != nil {
			return nil, err
		}
		cachedConfig = cfg
	}
	return cachedConfig, nil
}

func lockDir() string {
	return filepath.Join(getDataDir(), "locks")
}
