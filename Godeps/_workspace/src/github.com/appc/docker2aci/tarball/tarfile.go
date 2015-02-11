package tarball

import (
	"archive/tar"
	"io"
)

// TarFile is a representation of a file in a tarball. It consists of two parts,
// the Header and the Stream. The Header is a regular tar header, the Stream
// is a byte stream that can be used to read the file's contents
type TarFile struct {
	Header    *tar.Header
	TarStream io.Reader
}

// Name returns the name of the file as reported by the header
func (t *TarFile) Name() string {
	return t.Header.Name
}

// Linkname returns the Linkname of the file as reported by the header
func (t *TarFile) Linkname() string {
	return t.Header.Linkname
}
