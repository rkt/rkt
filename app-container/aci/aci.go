package aci

// Package aci contains a small library to validate files that comply with the ACI spec

import (
	"archive/tar"
	"fmt"
	"io"
	"io/ioutil"
	"os"
)

// ValidateTar checks that a given io.Reader, which should represent a tar
// file, contains a directory layout which matches the ACI spec
func ValidateTar(r io.Reader) error {
	// TODO(jonboulle): do this in memory instead of writing out to disk? :/
	dir, err := ioutil.TempDir("", "aci-validator")
	if err != nil {
		return fmt.Errorf("error creating tempdir for RAF validation: %v", err)
	}
	defer os.RemoveAll(dir)
	t := tar.NewReader(r)
	err = ExtractTar(t, dir)
	if err != nil {
		return err
	}
	return ValidateLayout(dir)
}
