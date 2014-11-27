package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/coreos-inc/rkt/app-container/discovery"
)

var (
	cmdFetch = &Command{
		Name:        "fetch",
		Description: "Discover and download an app container image",
		Summary:     "Discover, download and store on disk the app container image for one or more apps",
		Run:         runFetch,
	}
)

func runFetch(args []string) (exit int) {
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "discover: at least one name required")
	}

	for _, name := range args {
		labels, err := appFromString(name)
		if err != nil {
			stderr("%s: %s", name, err)
			return 1
		}
		eps, err := discovery.DiscoverEndpoints(labels["name"], labels["ver"], labels["os"], labels["arch"], transportFlags.Insecure)
		if err != nil {
			stderr("error fetching %s: %s", name, err)
			return 1
		}
		// TODO(philips): store the images..
		fmt.Println(strings.Join(eps.Sig, ","))
		fmt.Println(strings.Join(eps.ACI, ","))
		fmt.Println(strings.Join(eps.Keys, ","))
	}

	return
}
