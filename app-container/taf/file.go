package taf

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os/exec"
)

type FileType string

const (
	TypeGzip    = FileType("gz")
	TypeBzip2   = FileType("bz2")
	TypeXz      = FileType("xz")
	TypeUnknown = FileType("unknown")

	readLen = 6 // bytes to sniff

	hexHdrGzip  = "1f8b"
	hexHdrBzip2 = "425a68"
	hexHdrXz    = "fd377a585a00"
)

var (
	hdrGzip  []byte
	hdrBzip2 []byte
	hdrXz    []byte
)

func init() {
	var err error
	hdrGzip, err = hex.DecodeString(hexHdrGzip)
	if err != nil {
		panic(err)
	}
	hdrBzip2, err = hex.DecodeString(hexHdrBzip2)
	if err != nil {
		panic(err)
	}
	hdrXz, err = hex.DecodeString(hexHdrXz)
	if err != nil {
		panic(err)
	}
}

// DetectType attempts to detect the type of file that the given
// reader represents by reading the first few bytes and comparing
// it against known file signatures (magic numbers)
func DetectFileType(r io.Reader) (FileType, error) {
	var b bytes.Buffer
	n, err := io.CopyN(&b, r, readLen)
	if err != nil {
		return TypeUnknown, err
	}
	if n != readLen {
		return TypeUnknown, fmt.Errorf("error reading first %d bytes", readLen)
	}
	switch {
	case bytes.HasPrefix(b.Bytes(), hdrGzip):
		return TypeGzip, nil
	case bytes.HasPrefix(b.Bytes(), hdrBzip2):
		return TypeBzip2, nil
	case bytes.HasPrefix(b.Bytes(), hdrXz):
		return TypeXz, nil
	default:
		return TypeUnknown, nil
	}
}

// XzReader shells out to a command line xz executable (if
// available) to decompress the given io.Reader using the xz
// compression format
func XzReader(r io.Reader) io.ReadCloser {
	rpipe, wpipe := io.Pipe()
	ex, err := exec.LookPath("xz")
	if err != nil {
		log.Fatalf("couldn't find xz executable: %v", err)
	}
	cmd := exec.Command(ex, "--decompress", "--stdout")
	cmd.Stdin = r
	cmd.Stdout = wpipe

	go func() {
		err := cmd.Run()
		wpipe.CloseWithError(err)
	}()

	return rpipe
}
