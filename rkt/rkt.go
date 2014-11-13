package main

//
// Rocket is a reference implementation of the app container specification.
//
// Execution on Rocket is divided into a number of stages, and the `rkt`
// binary implements the first stage (stage 0), which consists of the
// following tasks:
// - Generating the Container Unique ID (UID)
// - Generating the container document
// - Creating a directory for the container
// - Copying the stage1 into the container directory
// - Copying the RAFs for each app into the stage2 directories
//
// Given a run command such as:
//	rkt run --volume bind:/opt/tenant1/database \
//		example.com/data-downloader-1.0.0 \
//		example.com/ourapp-1.0.0 \
//		example.com/logbackup-1.0.0
//
// the container doc generated will be compliant with the ACE spec.
//

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

var (
	fs = flag.NewFlagSet("rkt", flag.ExitOnError)

	flagVolumes stringSlice
)

func init() {
	fs.Var(&flagVolumes, "volume", "volumes to mount into the shared container environment")
}

func main() {
	fs.Parse(os.Args[1:])
	args := fs.Args()
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "usage: rkt run [image...]\n")
		os.Exit(0)
	}
	cmd := args[0]
	switch cmd {
	case "run":
	default:
		fmt.Fprintf(os.Stderr, "rkt: unknown subcommand: %q\n", cmd)
		os.Exit(1)
	}
	fs.Parse(args[1:])
	images := fs.Args()
	fmt.Println("run rocket run")
	fmt.Printf("images: %s\n", images)
	fmt.Printf("volumes: %s\n", flagVolumes)
}

// stringSlice implements the flag.Value interface
type stringSlice []string

func (ss *stringSlice) Set(s string) error {
	// TODO(jonboulle): validate
	*ss = append(*ss, s)
	return nil
}

func (ss *stringSlice) String() string {
	return strings.Join(*ss, ",")
}
