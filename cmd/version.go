package main

import (
	"fmt"

	"github.com/coreos-inc/rkt/rkt"
)

var cmdVersion = &Command{
	Name:        "version",
	Description: "Print the version and exit",
	Summary:     "Print the version and exit",
	Run:         runVersion,
}

func runVersion(args []string) (exit int) {
	fmt.Printf("rkt version %s\n", rkt.Version)
	return
}
