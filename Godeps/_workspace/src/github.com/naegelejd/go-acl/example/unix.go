package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/coreos/rkt/Godeps/_workspace/src/github.com/naegelejd/go-acl"
)

func main() {
	flag.Parse()
	if flag.NArg() < 1 {
		log.Fatal("Missing filename")
	}
	filename := flag.Arg(0)

	a, err := acl.GetFileAccess(filename)
	if err != nil {
		log.Fatalf("Failed to get ACL from %s (%s)", filename, err)
	}
	defer a.Free()

	fmt.Print("ACL repr:\n", a)
}
