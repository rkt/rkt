package tarball

import (
	"archive/tar"
	"fmt"
	"io"
)

// WalkFunc is a func for handling each file (header and byte stream) in a tarball
type WalkFunc func(t *TarFile) error

// Walk walks through the files in the tarball represented by tarstream and
// passes each of them to the WalkFunc provided as an argument
func Walk(tarReader tar.Reader, walkFunc func(t *TarFile) error) error {
	for {
		hdr, err := tarReader.Next()
		if err == io.EOF {
			// end of tar archive
			break
		}
		if err != nil {
			return fmt.Errorf("Error reading tar entry: %v", err)
		}
		if err := walkFunc(&TarFile{Header: hdr, TarStream: &tarReader}); err != nil {
			return err
		}
	}
	return nil
}
