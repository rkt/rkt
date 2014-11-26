package main

import (
	"fmt"

	"github.com/coreos-inc/rkt/app-container/schema"
)

var cmdVersion = &Command{
	Name:        "version",
	Description: "Print the version and exit",
	Summary:     "Print the version and exit",
	Run:         runVersion,
}

func runVersion(args []string) (exit int) {
	fmt.Printf("acfmt version %s\n", schema.AppContainerVersion.String())
	return
}
