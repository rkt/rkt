package main

import (
	"archive/tar"
	"fmt"
	"os"
	"path/filepath"

	"github.com/containers/standard/taf"
	"github.com/coreos-inc/rkt/downloadstore"
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

	ds := downloadstore.NewDownloadStore(globalFlags.Dir)

	for _, img := range args {
		rem := downloadstore.NewRemote(img, []string{})
		err := ds.Get(rem)
		if err != nil && rem.File == "" {
			rem, err = rem.Download(*ds)
			if err != nil {
				fmt.Fprintf(os.Stderr, "downloading: %v\n", err)
			}
		}
		rs, err := ds.ObjectStream(rem.File)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error fetching: %v\n", err)
			return 1
		}

		tr := tar.NewReader(rs)
		dir := filepath.Join(root, "sha256-"+rem.File)
		if err := os.MkdirAll(dir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "fetch: error creating image directory: %v", err)
			return 1
		}
		if err := taf.ExtractTar(tr, dir); err != nil {
			fmt.Fprintf(os.Stderr, "fetch: error extracting tar: %v", err)
			return 1
		}
		fmt.Println(rem.File)
	}
	return
}
