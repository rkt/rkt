//+build !linux

package main

import "fmt"

var (
	cmdStatus = &Command{
		Name:    "status",
		Summary: "Check the status of a rkt job",
		Run:     runStatus,
	}
)

func runStatus(args []string) (exit int) {
	fmt.Println("status is not implemented on this platform.")
	return
}
