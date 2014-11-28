package main

import (
	"archive/tar"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/coreos-inc/rkt/app-container/aci"
	"github.com/coreos-inc/rkt/app-container/schema"
	"github.com/coreos-inc/rkt/pkg/tarheader"
)

var (
	buildFileSetName string
	buildAppManifest string
	buildRootfs      bool
	buildOverwrite   bool
	cmdBuild         = &Command{
		Name:        "build",
		Description: "Build a FileSet ACI from the target directory",
		Summary:     "Build a FileSet ACI from the target directory",
		Usage:       "[--overwrite] --name=NAME DIRECTORY OUTPUT_FILE",
		Run:         runBuild,
	}
)

func init() {
	cmdBuild.Flags.StringVar(&buildFileSetName, "fileset-name", "",
		"Build a FileSet Image, by this name (e.g. example.com/reduce-worker)")
	cmdBuild.Flags.StringVar(&buildAppManifest, "app-manifest", "",
		"Build an App Image with this App Manifest")
	cmdBuild.Flags.BoolVar(&buildRootfs, "rootfs", true,
		"Whether the supplied directory is a rootfs. If false, it will be assume the supplied directory already contains a rootfs/ subdirectory.")
	cmdBuild.Flags.BoolVar(&buildOverwrite, "overwrite", false, "Overwrite target file if it already exists")
}

func buildWalker(root string, aw aci.ArchiveWriter, rootfs bool) filepath.WalkFunc {
	return func(path string, info os.FileInfo, err error) error {
		relpath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if rootfs {
			if relpath == "." {
				relpath = ""
			}
			relpath = "rootfs/" + relpath
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
	switch {
	case buildFileSetName != "" && buildAppManifest == "":
	case buildFileSetName == "" && buildAppManifest != "":
	default:
		stderr("build: must specify either --fileset-name or --app-manifest")
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
	tr := tar.NewWriter(fh)

	var aw aci.ArchiveWriter
	if buildFileSetName != "" {
		aw, err = aci.NewFileSetWriter(buildFileSetName, tr)
		if err != nil {
			stderr("build: Unable to create FileSetWriter: %v", err)
			return 1
		}
	} else {
		b, err := ioutil.ReadFile(buildAppManifest)
		if err != nil {
			stderr("build: Unable to read App Manifest: %v", err)
			return 1
		}
		var am schema.AppManifest
		if err := am.UnmarshalJSON(b); err != nil {
			stderr("build: Unable to load App Manifest: %v", err)
			return 1
		}
		aw = aci.NewAppWriter(am, tr)
	}

	filepath.Walk(root, buildWalker(root, aw, buildRootfs))

	err = aw.Close()
	if err != nil {
		stderr("build: Unable to close FileSet image %s: %v", tgt, err)
		return 1
	}

	return
}
