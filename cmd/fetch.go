package main

import (
	"archive/tar"
	"compress/bzip2"
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/containers/standard/schema/types"
	"github.com/containers/standard/taf"
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
	for _, img := range args {
		h, err := types.NewHash(img)
		if err != nil {
			log.Fatalf("bad hash given: %v", err)
		}
		rs := fetchImage(h.String())
		typ, err := taf.DetectFileType(rs)
		if err != nil {
			log.Fatalf("error detecting image type: %v", err)
		}
		if _, err := rs.Seek(0, 0); err != nil {
			log.Fatalf("error seeking image: %v", err)
		}
		var r io.Reader
		switch typ {
		case taf.TypeGzip:
			r, err = gzip.NewReader(rs)
			if err != nil {
				log.Fatalf("error reading gzip: %v", err)
			}
		case taf.TypeBzip2:
			r = bzip2.NewReader(rs)
		case taf.TypeXz:
			r = taf.XzReader(rs)
		case taf.TypeUnknown:
			log.Fatalf("error: unknown image filetype")
		default:
			// should never happen
			panic("no type returned from DetectFileType?")
		}
		tr := tar.NewReader(r)
		dir := filepath.Join(root, h.String())
		if err := os.MkdirAll(dir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "fetch: error creating image directory: %v", err)
			return 1
		}
		taf.ExtractTar(tr, dir)
	}
	return
}

// fetchImage returns an io.ReadSeeker accessing the image of the given hash.
// Right now it simply pulls from a local file on disk, but in future it would
// retrieve from a web service or similar.
func fetchImage(hash string) io.ReadSeeker {
	fh, err := os.Open(hash)
	if err != nil {
		log.Fatalf("error opening image: %v", err)
	}
	return fh
}
