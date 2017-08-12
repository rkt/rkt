// Copyright 2017 The rkt Authors
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
	"io"
	"os"

	"github.com/spf13/cobra"
)

var (
	cmdCompletion = &cobra.Command{
		Use:   "completion SHELL",
		Short: "Output shell completion code for the specified shell",
		Long: `This command outputs completion code for the specified shell. The generated
code must be evaluated to provide interactive completion of rkt sub-commands.
This can be done by sourcing it from the .bash_profile.

Save completion code in a home directory and then include it in .bash_profile
script:

	$ rkt completion bash > $HOME/.rkt.bash.inc
	$ printf '
# rkt shell completion
source "$HOME/.rkt.bash.inc"
' >> $HOME/.bash_profile
	$ source $HOME/.bash_profile

Alternatively, include the completion code directly into the launched shell:

	$ source <(rkt completion bash)`,
		ValidArgs: []string{"bash"},
		Run:       runWrapper(newCompletion(os.Stdout)),
	}

	bashCompletionFunc = `__rkt_parse_image()
{
	local rkt_output
	if rkt_output=$(rkt image list --no-legend 2>/dev/null); then
		out=($(echo "${rkt_output}" | awk '{print $1}'))
		COMPREPLY=( $( compgen -W "${out[*]}" -- "$cur" ) )
	fi
}

__rkt_parse_list()
{
	local rkt_output
	if rkt_output=$(rkt list --no-legend 2>/dev/null); then
		if [[ -n "$1" ]]; then
			out=($(echo "${rkt_output}" | grep ${1} | awk '{print $1}'))
		else
			out=($(echo "${rkt_output}" | awk '{print $1}'))
		fi
		COMPREPLY=( $( compgen -W "${out[*]}" -- "$cur" ) )
	fi
}

__custom_func() {
	case ${last_command} in
		rkt_image_export | \
		rkt_image_extract | \
		rkt_image_cat-manifest | \
		rkt_image_render | \
		rkt_image_rm | \
		rkt_run | \
		rkt_prepare)
			__rkt_parse_image
			return
			;;
		rkt_run-prepared)
			__rkt_parse_list prepared
			return
			;;
		rkt_enter)
			__rkt_parse_list running
			return
			;;
		rkt_rm)
			__rkt_parse_list "prepare\|exited"
			return
			;;
		rkt_status)
			__rkt_parse_list
			return
			;;
		*)
			;;
	esac
}
`

	completionShells = map[string]func(io.Writer, *cobra.Command) error{
		"bash": runBashCompletion,
	}
)

func init() {
	cmdRkt.AddCommand(cmdCompletion)
}

// newCompletion creates a new command with a bounded writer. Writer
// is used to print the generated shell-completion script, which is
// intended to be consumed by the CLI users.
func newCompletion(w io.Writer) func(*cobra.Command, []string) int {
	return func(cmd *cobra.Command, args []string) int {
		return runCompletion(w, cmd, args)
	}
}

// runCompletion is a command handler to generate the shell script with
// shell completion functions.
//
// It ensures that there are enough arguments to generate the completion
// scripts.
func runCompletion(w io.Writer, cmd *cobra.Command, args []string) (exit int) {
	if len(args) == 0 {
		stderr.Print("shell type is not specified")
		return 254
	}

	if len(args) > 1 {
		stderr.Print("too many arguments, only shell type is expected")
		return 254
	}

	// Right now only bash completion is supported, but zsh could be
	// supported in a future as well.
	completion, ok := completionShells[args[0]]
	if !ok {
		stderr.Printf("'%s' shell is not supported", args[0])
		return 254
	}

	// Write the shell completion to the specified writer.
	err := completion(w, cmd.Parent())
	if err != nil {
		stderr.PrintE("completion failed", err)
		return 254
	}

	return 0
}

// runBashCompletion generates bash completion script by delegating
// this responsibility to the cobra package itself.
func runBashCompletion(out io.Writer, cmd *cobra.Command) error {
	return cmd.GenBashCompletion(out)
}
