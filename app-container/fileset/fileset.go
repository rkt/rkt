package fileset

import (
	"archive/tar"
	"bytes"
	"errors"
	"fmt"
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

func (aw *ArchiveWriter) AddFile(path string, hdr *tar.Header, r io.Reader)  error {
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
		Name: "fileset",
		Mode: 0655,
		Uid: 0,
		Gid: 0,
		Size: int64(buf.Len()),
		ModTime: now,
		Typeflag: tar.TypeReg,
		Uname: "root",
		Gname: "root",
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
