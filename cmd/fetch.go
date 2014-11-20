package main

import "fmt"

var (
	cmdFetch = &Command{
		Name:    "fetch",
		Summary: "Fetch image(s) from a server",
		Usage:   "IMAGE_URL...",
		Run:     runFetch,
	}
)

func runFetch(args []string) (exit int) {
	fmt.Println("Not implemented.")
	return
}
