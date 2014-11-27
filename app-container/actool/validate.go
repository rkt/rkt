package main

import (
	"compress/bzip2"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/coreos-inc/rkt/app-container/aci"
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
		stderr("must pass one or more files")
		return 1
	}

	for _, path := range args {
		file, err := os.OpenFile(path, os.O_RDONLY, 0666)
		if err != nil {
			stderr("unable to open %s: %v\n", path, err)
			return 1
		}

		// First key off extension, and then possibly fall back to
		switch filepath.Ext(path) {
		case schema.ACIExtension:
			err := validateACI(file)
			file.Close()
			if err != nil {
				stderr("%s: error validating: %v\n", path, err)
				return 1
			}
			if globalFlags.Debug {
				stderr("%s: valid aci\n", path)
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

func validateACI(rs io.ReadSeeker) error {
	typ, err := aci.DetectFileType(rs)
	if err != nil {
		return err
	}
	if _, err := rs.Seek(0, 0); err != nil {
		return err
	}
	var r io.Reader
	switch typ {
	case aci.TypeGzip:
		r, err = gzip.NewReader(rs)
		if err != nil {
			return fmt.Errorf("error reading gzip: %v", err)
		}
	case aci.TypeBzip2:
		r = bzip2.NewReader(rs)
	case aci.TypeXz:
		r = aci.XzReader(rs)
	case aci.TypeUnknown:
		return errors.New("unknown filetype")
	default:
		// should never happen
		panic("no type returned from DetectFileType?")
	}
	return aci.ValidateTar(r)
}
