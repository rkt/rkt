package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/coreos/rocket/app-container/discovery"
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
		// TODO(philips): store the images..
		fmt.Println(strings.Join(eps.Sig, ","))
		fmt.Println(strings.Join(eps.ACI, ","))
		fmt.Println(strings.Join(eps.Keys, ","))
	}

	return
}
