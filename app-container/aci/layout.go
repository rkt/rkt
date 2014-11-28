package aci

/*

Image Layout

The on-disk layout of an app container is straightforward. It includes a rootfs with all of the files that will exist in the root of the app and an app manifest describing how to execute the app.

/app
/rootfs
/rootfs/usr/bin/mysql
/rootfs/....

*/

import (
	"archive/tar"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/coreos-inc/rkt/app-container/schema"
)

var (
	ErrNoRootFS   = errors.New("no rootfs found in layout")
	ErrNoManifest = errors.New("no app or fileset manifest found in layout")
)

// ValidateLayout takes a directory and validates that the layout of the directory
// matches that expected by the Application Container Image format.
// If any errors are encountered during the validation, it will abort and
// return the first one.
func ValidateLayout(dir string) error {
	fi, err := os.Stat(dir)
	if err != nil {
		return fmt.Errorf("error accessing layout: %v", err)
	}
	if !fi.IsDir() {
		return fmt.Errorf("given path %q is not a directory", dir)
	}
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("error reading directory: %v", err)
	}
	var rfs, am, fsm string
	for _, f := range files {
		name := f.Name()
		fpath := filepath.Join(dir, name)
		switch name {
		case "app":
			am = fpath
		case "fileset":
			fsm = fpath
		case "rootfs":
			rfs = fpath
		default:
			return fmt.Errorf("unrecognized file path in layout: %q", fpath)
		}
	}
	if am == "" && fsm == "" {
		return ErrNoManifest
	}
	if am != "" && rfs == "" {
		return ErrNoRootFS
	}
	if err := validateAppManifest(am); err != nil {
		return err
	}
	return nil
}

// ValidateLayout takes a *tar.Reader and validates that the layout of the
// filesystem the reader encapsulates matches that expected by the
// Application Container Image format.  If any errors are encountered during
// the validation, it will abort and return the first one.
func ValidateArchive(tr *tar.Reader) error {
	var flist []string
	manifestOK := false
	fsm := &schema.FileSetManifest{}
	for {
		hdr, err := tr.Next()
		if err != nil {
			switch {
			case err == io.EOF && manifestOK:
				err = filesEqual(fsm.Files, flist)
				if err != nil {
					return err
				}
				return nil
			case !manifestOK:
				return errors.New("fileset: missing fileset manifest")
			default:
				return err
			}
		}

		flist = append(flist, hdr.Name)

		switch hdr.Name {
		case "fileset":
			var b bytes.Buffer
			_, err = io.Copy(&b, tr)
			if err != nil {
				return err
			}
			err = fsm.UnmarshalJSON(b.Bytes())
			if err != nil {
				return err
			}
			manifestOK = true
		case "app":
		}
	}
	return nil
}

// validateAppManifest ensures that the file at the given path is a valid
// AppManifest.
func validateAppManifest(fpath string) error {
	f, err := os.Open(fpath)
	if err != nil {
		return fmt.Errorf("error opening app manifest: %v", err)
	}
	b, err := ioutil.ReadAll(f)
	if err != nil {
		return fmt.Errorf("error reading app manifest: %v", err)
	}
	var am schema.AppManifest
	if err = json.Unmarshal(b, &am); err != nil {
		return fmt.Errorf("error unmarshaling app manifest: %v", err)
	}
	return nil
}

func filesEqual(a, b []string) error {
	if len(a) != len(b) {
		return errors.New("different file count")
	}

	for i := range a {
		if a[i] != b[i] {
			return fmt.Errorf("file mismatch %s != %s", a[i], b[i])
		}
	}
	return nil
}
