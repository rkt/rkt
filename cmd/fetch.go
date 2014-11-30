package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/coreos/rocket/cas"
)

const (
	imgDir = "images"
)

var (
	cmdFetch = &Command{
		Name:    "fetch",
		Summary: "Fetch image(s) and store them in the local cache",
		Usage:   "IMAGE_URL...",
		Run:     runFetch,
	}
)

func fetchURL(img string, ds *cas.Store) (string, error) {
	rem := cas.NewRemote(img, []string{})
	err := ds.ReadIndex(rem)
	if err != nil && rem.Blob == "" {
		rem, err = rem.Download(*ds)
		if err != nil {
			return "", fmt.Errorf("downloading: %v\n", err)
		}
	}
	return rem.Blob, nil
}

func runFetch(args []string) (exit int) {
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "fetch: Must provide at least one image\n")
		return 1
	}
	root := filepath.Join(globalFlags.Dir, imgDir)
	if err := os.MkdirAll(root, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "fetch: error creating image directory: %v", err)
		return 1
	}

	ds := cas.NewStore(globalFlags.Dir)

	for _, img := range args {
		hash, err := fetchURL(img, ds)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v", err)
			return 1
		}
		fmt.Println(hash)
	}

	return
}
