package aci

// Package aci contains a small library to validate files that comply with the ACI spec

import (
	"archive/tar"
	"fmt"
	"io/ioutil"
	"os"

	ptar "github.com/coreos-inc/rkt/pkg/tar"
)

// ValidateTar checks that a given tar.Reader contains a directory layout which
// matches the ACI spec
func ValidateTar(tr *tar.Reader) error {
	// TODO(jonboulle): do this in memory instead of writing out to disk? :/
	dir, err := ioutil.TempDir("", "aci-validator")
	if err != nil {
		return fmt.Errorf("error creating tempdir for RAF validation: %v", err)
	}
	defer os.RemoveAll(dir)
	err = ptar.ExtractTar(tr, dir)
	if err != nil {
		return err
	}
	return ValidateLayout(dir)
}
