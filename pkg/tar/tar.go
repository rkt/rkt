// Copyright 2014 CoreOS, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tar

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

const DEFAULT_DIR_MODE os.FileMode = 0755

type insecureLinkError error

// Map of paths that should be whitelisted. The paths should be relative to the
// root of the tar file and should be cleaned (for example using filepath.Clean)
type PathWhitelistMap map[string]struct{}

// ExtractTar extracts a tarball (from a tar.Reader) into the given directory
// if pwl is not nil, only the paths in the map are extracted.
func ExtractTar(tr *tar.Reader, dir string, pwl PathWhitelistMap) error {
	um := syscall.Umask(0)
	defer syscall.Umask(um)
	for {
		hdr, err := tr.Next()
		switch err {
		case io.EOF:
			return nil
		case nil:
			if pwl != nil {
				relpath := filepath.Clean(hdr.Name)
				if _, ok := pwl[relpath]; !ok {
					continue
				}
			}
			err = ExtractFile(tr, hdr, dir)
			if err != nil {
				return fmt.Errorf("error extracting tarball: %v", err)
			}
		default:
			return fmt.Errorf("error extracting tarball: %v", err)
		}
	}
}

// Extract the file provided by hdr
func ExtractFile(tr *tar.Reader, hdr *tar.Header, dir string) error {
	p := filepath.Join(dir, hdr.Name)
	fi := hdr.FileInfo()
	typ := hdr.Typeflag

	// Create parent dir if it doesn't exists
	if err := os.MkdirAll(filepath.Dir(p), DEFAULT_DIR_MODE); err != nil {
		return err
	}
	switch {
	case typ == tar.TypeReg || typ == tar.TypeRegA:
		if err := os.MkdirAll(filepath.Dir(p), DEFAULT_DIR_MODE); err != nil {
			return err
		}
		f, err := os.OpenFile(p, os.O_CREATE|os.O_RDWR, fi.Mode())
		if err != nil {
			return err
		}
		_, err = io.Copy(f, tr)
		if err != nil {
			f.Close()
			return err
		}
		f.Close()
	case typ == tar.TypeDir:
		if err := os.MkdirAll(p, fi.Mode()); err != nil {
			return err
		}
		dir, err := os.Open(p)
		if err != nil {
			return err
		}
		if err := dir.Chmod(fi.Mode()); err != nil {
			dir.Close()
			return err
		}
		dir.Close()
	case typ == tar.TypeLink:
		dest := filepath.Join(dir, hdr.Linkname)
		if !strings.HasPrefix(dest, dir) {
			return insecureLinkError(fmt.Errorf("insecure link %q -> %q", p, hdr.Linkname))
		}
		if err := os.Link(dest, p); err != nil {
			return err
		}
	case typ == tar.TypeSymlink:
		dest := filepath.Join(filepath.Dir(p), hdr.Linkname)
		if !strings.HasPrefix(dest, dir) {
			return insecureLinkError(fmt.Errorf("insecure symlink %q -> %q", p, hdr.Linkname))
		}
		if err := os.Symlink(hdr.Linkname, p); err != nil {
			return err
		}
	case typ == tar.TypeChar:
		dev := makedev(int(hdr.Devmajor), int(hdr.Devminor))
		mode := uint32(fi.Mode()) | syscall.S_IFCHR
		if err := syscall.Mknod(p, mode, dev); err != nil {
			return err
		}
	case typ == tar.TypeBlock:
		dev := makedev(int(hdr.Devmajor), int(hdr.Devminor))
		mode := uint32(fi.Mode()) | syscall.S_IFBLK
		if err := syscall.Mknod(p, mode, dev); err != nil {
			return err
		}
	// TODO(jonboulle): implement other modes
	default:
		return fmt.Errorf("unsupported type: %v", typ)
	}

	return nil

}

// makedev mimics glib's gnu_dev_makedev
func makedev(major, minor int) int {
	return (minor & 0xff) | (major & 0xfff << 8) | int((uint64(minor & ^0xff) << 12)) | int(uint64(major & ^0xfff)<<32)
}
