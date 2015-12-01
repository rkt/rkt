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

package flag

import (
	"fmt"
	"strconv"

	"github.com/coreos/rkt/Godeps/_workspace/src/github.com/spf13/pflag"
)

// TODO: Remove this file before 1.0 release

// flagInsecureSkipVerify is a deprecated flag that is equivalent to setting
// "--insecure-options" to 'all' when true and 'none' when false.
type skipVerify struct {
	flags *SecFlags
}

func InstallDeprecatedSkipVerify(flagset *pflag.FlagSet, flags *SecFlags) {
	sv := &skipVerify{
		flags: flags,
	}
	svFlag := flagset.VarPF(sv, "insecure-skip-verify", "", "DEPRECATED")
	svFlag.DefValue = "false"
	svFlag.NoOptDefVal = "true"
	svFlag.Hidden = true
	svFlag.Deprecated = "please use --insecure-options."
}

func (sv *skipVerify) Set(s string) error {
	v, err := strconv.ParseBool(s)
	if err != nil {
		return err
	}

	switch v {
	case false:
		err = sv.flags.Set("none")
	case true:
		err = sv.flags.Set("all")
	}
	return err
}

func (sv *skipVerify) String() string {
	return fmt.Sprintf("%v", *sv)
}

func (sv *skipVerify) Type() string {
	// Must return "bool" in order to place naked flag before subcommand.
	// For example, rkt --insecure-skip-verify run docker://image
	return "bool"
}
