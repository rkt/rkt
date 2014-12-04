//+build linux

package main

import "fmt"

var (
	cmdStatus = &Command{
		Name:    "status",
		Summary: "Check the status of a rkt job",
		Usage:   "UUID",
		Run:     runStatus,
	}
)

func runStatus(args []string) (exit int) {
	fmt.Println("Not implemented.")
	return
}
