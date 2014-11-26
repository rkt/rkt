package main

import (
	"archive/tar"
	"fmt"
	"os"
	"path/filepath"

	"github.com/coreos-inc/rkt/app-container/fileset"
	"github.com/coreos-inc/rkt/app-container/schema"
	"github.com/coreos-inc/rkt/pkg/tarheader"
)

var (
	buildName      string
	buildOverwrite bool
	cmdBuild       = &Command{
		Name:        "build",
		Description: "Build a fileset from the target directory",
		Summary:     "Build a fileset from the target directory",
		Run:         runBuild,
	}
)

func init() {
	cmdBuild.Flags.StringVar(&buildName, "name", "",
		"Name of the fileset (e.g. example.com/reduce-worker)")
	cmdBuild.Flags.BoolVar(&buildOverwrite, "overwrite", false, "Overwrite target file if it already exists")
}

func buildWalker(root string, aw *fileset.ArchiveWriter) filepath.WalkFunc {
	return func(path string, info os.FileInfo, err error) error {
		relpath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if relpath == "." {
			return nil
		}

		link := ""
		var file *os.File
		switch info.Mode() & os.ModeType {
		default:
			file, err = os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()
		case os.ModeSymlink:
			target, err := os.Readlink(path)
			if err != nil {
				return err
			}
			link = target
		}

		hdr, err := tar.FileInfoHeader(info, link)
		if err != nil {
			panic(err)
		}
		tarheader.Populate(hdr, info)
		aw.AddFile(relpath, hdr, file)

		return nil
	}
}

func runBuild(args []string) (exit int) {
	if len(args) != 2 {
		fmt.Fprintf(os.Stderr, "build: Must provide directory and target fileset\n")
		return 1
	}

	root := args[0]
	tgt := args[1]
	ext := filepath.Ext(tgt)
	if ext != schema.FileSetExtension {
		fmt.Fprintf(os.Stderr, "fileset: Extension must be %s was %s\n", schema.FileSetExtension, ext)
	}

	fsm := schema.NewFileSetManifest(buildName)

	mode := os.O_CREATE | os.O_WRONLY
	if !buildOverwrite {
		mode |= os.O_EXCL
	}
	afs, err := os.OpenFile(tgt, mode, 0655)
	if err != nil {
		if os.IsExist(err) {
			fmt.Fprintf(os.Stderr, "Target file exists (try --overwrite)\n")
		} else {
			fmt.Fprintf(os.Stderr, "fileset: Unable to open target %s: %v\n", tgt, err)
		}
		return 1
	}
	w := tar.NewWriter(afs)
	aw := fileset.NewArchiveWriter(*fsm, w)
	filepath.Walk(root, buildWalker(root, aw))

	err = aw.Close()
	if err != nil {
		fmt.Fprintf(os.Stderr, "fileset: Unable to close fileset %s: %v\n", tgt, err)
		return 1
	}

	return
}
