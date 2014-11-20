package main

import "fmt"

var (
	cmdGC = &Command{
		Name:    "gc",
		Summary: "Garbage-collect rkt containers no longer in use",
		Usage:   "",
		Run:     runGC,
	}
)

func runGC(args []string) (exit int) {
	fmt.Println("Not implemented.")
	return
}
