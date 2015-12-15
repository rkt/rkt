package os

import (
	"fmt"
	"os"
	"syscall"
)

func Owner(fname string) (owner, group int, err error) {
	var f *os.File
	f, err = os.Open(fname)
	if err != nil {
		return
	}
	defer f.Close()

	g := &File{*f}
	return g.Owner()
}

type File struct{ os.File }

func (f *File) Owner() (owner, group int, err error) {
	var fi os.FileInfo
	fi, err = f.Stat()
	if err != nil {
		return
	}

	sys, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		err = fmt.Errorf("could not stat file")
	}

	return int(sys.Uid), int(sys.Gid), nil
}
