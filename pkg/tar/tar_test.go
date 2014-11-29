package tar

import (
	"archive/tar"
	"io"
	"io/ioutil"
	"os"
	"testing"
)

type testTarEntry struct {
	header   *tar.Header
	contents string
}

func newTestTar(entries []*testTarEntry) (string, error) {
	t, err := ioutil.TempFile("", "test-tar")
	if err != nil {
		return "", err
	}
	defer t.Close()
	tw := tar.NewWriter(t)
	for _, entry := range entries {
		if err := tw.WriteHeader(entry.header); err != nil {
			return "", err
		}
		if _, err := io.WriteString(tw, entry.contents); err != nil {
			return "", err
		}
	}
	if err := tw.Close(); err != nil {
		return "", err
	}
	return t.Name(), nil
}

func TestExtractTarInsecureSymlink(t *testing.T) {
	entries := []*testTarEntry{
		{
			contents: "hello",
			header: &tar.Header{
				Name: "hello.txt",
				Size: 5,
			},
		},
		{
			header: &tar.Header{
				Name:     "link.txt",
				Linkname: "hello.txt",
				Typeflag: tar.TypeSymlink,
			},
		},
	}
	insecureSymlinkEntries := append(entries, &testTarEntry{
		header: &tar.Header{
			Name:     "../etc/secret.conf",
			Linkname: "secret.conf",
			Typeflag: tar.TypeSymlink,
		},
	})
	insecureHardlinkEntries := append(entries, &testTarEntry{
		header: &tar.Header{
			Name:     "../etc/secret.conf",
			Linkname: "secret.conf",
			Typeflag: tar.TypeLink,
		},
	})
	for _, entries := range [][]*testTarEntry{insecureSymlinkEntries, insecureHardlinkEntries} {
		testTarPath, err := newTestTar(entries)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		defer os.Remove(testTarPath)
		containerTar, err := os.Open(testTarPath)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		tr := tar.NewReader(containerTar)
		tmpdir, err := ioutil.TempDir("", "rocket-temp-dir")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		os.RemoveAll(tmpdir)
		err = os.MkdirAll(tmpdir, 0755)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		err = ExtractTar(tr, tmpdir)
		if _, ok := err.(insecureLinkError); !ok {
			t.Errorf("expected insecureSymlinkError error")
		}
	}
}
