package fileset

import (
	"archive/tar"
	"bytes"
	"io"
	"time"

	"github.com/coreos-inc/rkt/app-container/schema"
)

type ArchiveWriter struct {
	*tar.Writer
	manifest *schema.FileSetManifest
}

func NewArchiveWriter(fsm schema.FileSetManifest, w *tar.Writer) *ArchiveWriter {
	aw := &ArchiveWriter{
		w,
		&fsm,
	}
	return aw
}

func (aw *ArchiveWriter) AddFile(path string, hdr *tar.Header, r io.Reader) error {
	aw.manifest.Files = append(aw.manifest.Files, path)
	err := aw.Writer.WriteHeader(hdr)
	if err != nil {
		return err
	}

	if r != nil {
		_, err := io.Copy(aw.Writer, r)
		if err != nil {
			return err
		}
	}

	return nil
}

func (aw *ArchiveWriter) Close() error {
	aw.manifest.Files = append(aw.manifest.Files, "fileset")
	out, err := aw.manifest.MarshalJSON()
	if err != nil {
		return err
	}
	buf := bytes.NewBuffer(out)
	now := time.Now()
	hdr := tar.Header{
		Name:       "fileset",
		Mode:       0655,
		Uid:        0,
		Gid:        0,
		Size:       int64(buf.Len()),
		ModTime:    now,
		Typeflag:   tar.TypeReg,
		Uname:      "root",
		Gname:      "root",
		ChangeTime: now,
	}
	err = aw.AddFile("fileset", &hdr, buf)
	if err != nil {
		return err
	}
	err = aw.Writer.Close()
	if err != nil {
		return err
	}
	return nil
}
