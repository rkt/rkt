//+build !linux

package main

import "fmt"

var (
	cmdRun = &Command{
		Name:    "run",
		Summary: "Run image(s) in an application container in rocket",
		Run:     runRun,
	}
)

func runRun(args []string) (exit int) {
	fmt.Println("run is not implemented on this platform.")
	return
}
