package main

import (
	"fmt"
	"strings"

	"github.com/coreos/rocket/Godeps/_workspace/src/github.com/appc/spec/discovery"
)

var (
	cmdDiscover = &Command{
		Name:        "discover",
		Description: "Discover the download URLs for an app",
		Summary:     "Discover the download URLs for one or more app container images",
		Usage:       "APP...",
		Run:         runDiscover,
	}
)

func init() {
	cmdDiscover.Flags.BoolVar(&transportFlags.Insecure, "insecure", false,
		"Allow insecure non-TLS downloads over http")
}

func runDiscover(args []string) (exit int) {
	if len(args) < 1 {
		stderr("discover: at least one name required")
	}

	for _, name := range args {
		app, err := discovery.NewAppFromString(name)
		if err != nil {
			stderr("%s: %s", name, err)
			return 1
		}
		eps, err := discovery.DiscoverEndpoints(*app, transportFlags.Insecure)
		if err != nil {
			stderr("error fetching %s: %s", name, err)
			return 1
		}
		for _, aciEndpoint := range eps.ACIEndpoints {
			fmt.Println("ACI: %s, Sig: %s\n", aciEndpoint.ACI, aciEndpoint.Sig)
		}
		if len(eps.Keys) > 0 {
			fmt.Println("Keys: " + strings.Join(eps.Keys, ","))
		}
	}

	return
}
