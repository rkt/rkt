package aci

/*

Image Layout

The on-disk layout of an app container is straightforward.
It includes a rootfs with all of the files that will exist in the root of the app and an app manifest describing how to execute the app.
The layout must contain either an app manifest, an app manifest and a fileset manifest, or a fileset only.
In the latter case, the layout/image is known as a "fileset image".

/app
/rootfs/
/rootfs/usr/bin/mysql

/app
/fileset
/rootfs/bin/httpd
/rootfs/config


/fileset
/rootfs/
/rootfs/bin/bash

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
	"strings"

	"github.com/coreos/rocket/app-container/schema"
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
	var flist []string
	var amOK, fsmOK, rfsOK bool
	var am, fsm io.Reader
	walkLayout := func(fpath string, fi os.FileInfo, err error) error {
		fpath = strings.TrimPrefix(fpath, dir)
		name := filepath.Base(fpath)
		switch name {
		case ".":
		case "app":
			am, err = os.Open(fpath)
			if err != nil {
				return err
			}
			amOK = true
		case "fileset":
			fsm, err = os.Open(fpath)
			if err != nil {
				return err
			}
			fsmOK = true
		case "rootfs":
			if !fi.IsDir() {
				return errors.New("rootfs is not a directory")
			}
			rfsOK = true
		default:
			flist = append(flist, fpath)
		}
		return nil
	}
	if err := filepath.Walk(dir, walkLayout); err != nil {
		return err
	}
	return validate(amOK, am, fsmOK, fsm, rfsOK, flist)
}

// ValidateLayout takes a *tar.Reader and validates that the layout of the
// filesystem the reader encapsulates matches that expected by the
// Application Container Image format.  If any errors are encountered during
// the validation, it will abort and return the first one.
func ValidateArchive(tr *tar.Reader) error {
	var flist []string
	var amOK, fsmOK, rfsOK bool
	var fsm, am bytes.Buffer
Tar:
	for {
		hdr, err := tr.Next()
		switch {
		case err == nil:
		case err == io.EOF:
			break Tar
		default:
			return err
		}
		switch hdr.Name {
		case "fileset":
			_, err := io.Copy(&fsm, tr)
			if err != nil {
				return err
			}
			fsmOK = true
		case "app":
			_, err := io.Copy(&am, tr)
			if err != nil {
				return err
			}
			amOK = true
		case "rootfs/":
			if !hdr.FileInfo().IsDir() {
				return fmt.Errorf("rootfs is not a directory")
			}
			rfsOK = true
		default:
			flist = append(flist, hdr.Name)
		}
	}
	return validate(amOK, &am, fsmOK, &fsm, rfsOK, flist)
}

// TODO(jonboulle): find a cleaner way to communicate instead of all these args.
func validate(amOK bool, am io.Reader, fsmOK bool, fsm io.Reader, rfsOK bool, files []string) error {
	if !amOK && !fsmOK {
		return ErrNoManifest
	}
	if amOK {
		if !rfsOK {
			return ErrNoRootFS
		}
		b, err := ioutil.ReadAll(am)
		if err != nil {
			return fmt.Errorf("error reading app manifest: %v", err)
		}
		var a schema.AppManifest
		if err := a.UnmarshalJSON(b); err != nil {
			return fmt.Errorf("app manifest validation failed: %v", err)
		}
	}
	var rfsfiles []string
	for _, f := range files {
		switch {
		case strings.HasPrefix(f, "rootfs"):
			rfsfiles = append(rfsfiles, strings.TrimPrefix(f, "rootfs"))
		default:
			return fmt.Errorf("unrecognized file path in layout: %q", f)
		}
	}
	if fsmOK {
		b, err := ioutil.ReadAll(fsm)
		if err != nil {
			return fmt.Errorf("error reading fileset manifest: %v", err)
		}
		var f schema.FilesetManifest
		if err := f.UnmarshalJSON(b); err != nil {
			return fmt.Errorf("fileset manifest validation failed: %v", err)
		}
		// TODO(jonboulle): this is not quite correct since it does not
		// deal with dependent filesets. Maybe filesAreSuperset()?
		return filesEqual(f.Files, rfsfiles)
	}
	return nil
}

// validateAppManifest ensures that the given io.Reader represents a valid
// AppManifest.
func validateAppManifest(r io.Reader) error {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return fmt.Errorf("error reading app manifest: %v", err)
	}
	var am schema.AppManifest
	if err = json.Unmarshal(b, &am); err != nil {
		return fmt.Errorf("error unmarshaling app manifest: %v", err)
	}
	return nil
}

// validateFilesetManifest ensures that the given io.Reader represents a valid
// FilesetManifest.
func validateFilesetManifest(r io.Reader) error {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return fmt.Errorf("error reading app manifest: %v", err)
	}
	var am schema.FilesetManifest
	if err = json.Unmarshal(b, &am); err != nil {
		return fmt.Errorf("error unmarshaling app manifest: %v", err)
	}
	return nil
}

func filesEqual(a, b []string) error {
	na := len(a)
	nb := len(b)
	if na != nb {
		return fmt.Errorf("fileset has different filecount to rootfs (%d != %d)", na, nb)
	}

	for i := range a {
		if a[i] != b[i] {
			return fmt.Errorf("file mismatch %s != %s", a[i], b[i])
		}
	}
	return nil
}
