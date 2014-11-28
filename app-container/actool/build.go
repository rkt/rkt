package main

import (
	"archive/tar"
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
		Description: "Build a FileSet ACI from the target directory",
		Summary:     "Build a FileSet ACI from the target directory",
		Usage:       "[--overwrite] --name=NAME DIRECTORY OUTPUT_FILE",
		Run:         runBuild,
	}
)

func init() {
	cmdBuild.Flags.StringVar(&buildName, "name", "",
		"Name of the FileSet (e.g. example.com/reduce-worker)")
	cmdBuild.Flags.BoolVar(&buildOverwrite, "overwrite", false, "Overwrite target file if it already exists")
}

func buildWalker(root string, aw fileset.ArchiveWriter) filepath.WalkFunc {
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
		// Because os.FileInfo's Name method returns only the base
		// name of the file it describes, it may be necessary to
		// modify the Name field of the returned header to provide the
		// full path name of the file.
		hdr.Name = relpath
		tarheader.Populate(hdr, info)
		aw.AddFile(relpath, hdr, file)

		return nil
	}
}

func runBuild(args []string) (exit int) {
	if len(args) != 2 {
		stderr("build: Must provide directory and output file")
		return 1
	}
	if buildName == "" {
		stderr("build: FileSet name cannot be empty")
		return 1
	}

	root := args[0]
	tgt := args[1]
	ext := filepath.Ext(tgt)
	if ext != schema.ACIExtension {
		stderr("build: Extension must be %s (given %s)", schema.ACIExtension, ext)
		return 1
	}

	mode := os.O_CREATE | os.O_WRONLY
	if !buildOverwrite {
		mode |= os.O_EXCL
	}
	fh, err := os.OpenFile(tgt, mode, 0655)
	if err != nil {
		if os.IsExist(err) {
			stderr("build: Target file exists (try --overwrite)")
		} else {
			stderr("build: Unable to open target %s: %v", tgt, err)
		}
		return 1
	}

	aw, err := fileset.NewFileSetWriter(buildName, tar.NewWriter(fh))
	if err != nil {
		stderr("build: Unable to create FileSetWriter: %v", err)
	}
	filepath.Walk(root, buildWalker(root, aw))

	err = aw.Close()
	if err != nil {
		stderr("build: Unable to close FileSet image %s: %v", tgt, err)
		return 1
	}

	return
}
