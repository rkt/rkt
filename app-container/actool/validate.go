package main

import (
	"archive/tar"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/coreos-inc/rkt/app-container/fileset"
	"github.com/coreos-inc/rkt/app-container/schema"
)

var cmdValidate = &Command{
	Name:        "validate",
	Description: "Validate an AppContainer file",
	Summary:     "Validate an AppContainer file",
	Run:         runValidate,
}

func runValidate(args []string) (exit int) {
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "must pass one or more files")
		return 1
	}

	for _, path := range args {
		file, err := os.OpenFile(path, os.O_RDONLY, 0666)
		if err != nil {
			fmt.Fprintf(os.Stderr, "unable to open %s: %v\n", path, err)
			return 1
		}

		switch filepath.Ext(path) {
		case schema.FileSetExtension:
			tr := tar.NewReader(file)
			err := fileset.ValidateArchive(tr)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s: error validating: %v\n", path, err)
				return 1
			}
			if globalFlags.Debug {
				fmt.Fprintf(os.Stderr, "%s: valid fileset\n", path)
			}
			continue
		}

		b, err := ioutil.ReadFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: unable to read file %s\n", path, err)
			return 1
		}

		typ := http.DetectContentType(b)
		switch {
		case typ == "text/plain; charset=utf-8":
			// TODO(philips): validate schema, need a package to take a ACKind
			// and lookup the validator
			k := schema.Kind{}
			k.UnmarshalJSON(b)
			if globalFlags.Debug {
				fmt.Fprintf(os.Stderr, "%s: valid %s\n", path, k.ACKind)
			}
		default:
			fmt.Fprintf(os.Stderr, "%s: unknown filetype %s\n", path, typ)
			return 1
		}
	}

	return
}
