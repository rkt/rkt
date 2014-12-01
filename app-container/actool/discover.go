package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/coreos/rocket/app-container/discovery"
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
		fmt.Fprintf(os.Stderr, "discover: at least one name required")
	}
	q := globalFlags.Quiet

	for _, name := range args {
		app, err := discovery.NewAppFromString(name)
		if err != nil {
			stderr(q, "%s: %s", name, err)
			return 1
		}
		eps, err := discovery.DiscoverEndpoints(*app, transportFlags.Insecure)

		if err != nil {
			stderr(q, "error fetching %s: %s", name, err)
			return 1
		}
		for _, list := range [][]string{eps.Sig, eps.ACI, eps.Keys} {
			if len(list) != 0 {
				fmt.Println(strings.Join(list, ","))
			}
		}
	}

	return
}
