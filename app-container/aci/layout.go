package aci

/*

Filesystem Layout

The on-disk format is straightforward, with a rootfs and an app manifest.

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
	"log"
	"os"
	"path/filepath"

	"github.com/coreos-inc/rkt/app-container/schema"
	"github.com/coreos-inc/rkt/app-container/schema/types"
)

var (
	ErrNoRootFS      = errors.New("no rootfs found in layout")
	ErrNoAppManifest = errors.New("no app manifest found in layout")
)

// ValidateLayout takes a directory and validates that the layout of the directory
// matches that expected by the Application Container Image format.
// If any errors are encountered during the validation, it will abort and
// return the first one.
// TODO(jonboulle): also support working on tar streams instead of requiring this be on disk
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
	var rootfs, manifest string
	for _, f := range files {
		name := f.Name()
		fpath := filepath.Join(dir, name)
		switch name {
		case "app":
			manifest = fpath
		case "rootfs":
			rootfs = fpath
		default:
			log.Printf("unrecognized file path %q - ignoring", fpath)
		}
	}
	if manifest == "" {
		return ErrNoAppManifest
	}
	if rootfs == "" {
		return ErrNoRootFS
	}
	rfsfiles, err := validateAppManifest(manifest)
	if err != nil {
		return err
	}
	if err := validateRootfs(rootfs, rfsfiles); err != nil {
		return err
	}
	return nil
}

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
// AppManifest. It returns a map of all files described in the manifest.
func validateAppManifest(fpath string) (map[string]types.File, error) {
	f, err := os.Open(fpath)
	if err != nil {
		return nil, fmt.Errorf("error opening app manifest: %v", err)
	}
	b, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("error reading app manifest: %v", err)
	}
	var am schema.AppManifest
	if err = json.Unmarshal(b, &am); err != nil {
		return nil, fmt.Errorf("error unmarshaling app manifest: %v", err)
	}
	return am.Files, nil
}

// validateRootfs ensures that a given rootfs filesystem is valid.
func validateRootfs(dir string, files map[string]types.File) error {
	fi, err := os.Stat(dir)
	if err != nil {
		return fmt.Errorf("error accessing rootfs: %v", err)
	}
	if !fi.IsDir() {
		return fmt.Errorf("rootfs is not a directory")
	}
	for path, file := range files {
		if err := validateFile(path, file); err != nil {
			return err
		}
	}
	return nil
}

func validateFile(path string, file types.File) error {
	// TODO(jonboulle): implement me
	// validate that the file matches the expected from the tarball?
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
