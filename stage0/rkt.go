package main

//
// Rocket is a reference implementation of the app container specification.
//
// Execution on Rocket is divided into a number of stages, and the `rkt`
// binary implements the first stage (stage 0), which consists of the
// following tasks:
// - Generating a Container Unique ID (UID)
// - Generating a Container Runtime Manifest
// - Creating a filesystem for the container
// - Setting up stage 1 and stage 2 directories in the filesystem
// - Copying the stage1 binary into the container filesystem
// - Fetching the specified application TAFs
// - Unpacking the TAFs and copying the RAF for each app into the stage2
//
// Given a run command such as:
//	rkt run --volume bind:/opt/tenant1/database \
//		example.com/data-downloader-1.0.0 \
//		example.com/ourapp-1.0.0 \
//		example.com/logbackup-1.0.0
//
// the container manifest generated will be compliant with the ACE spec.
//

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	// WARNING: here be dragons
	// TODO(jonboulle): vendor this once the schema is stable
	"github.com/containers/standard/schema"
	"github.com/containers/standard/schema/types"
	"github.com/containers/standard/taf"
	"github.com/coreos-inc/rkt/rkt"
)

var (
	fs = flag.NewFlagSet("rkt", flag.ExitOnError)

	flagDir     string
	flagStage1  string
	flagVolumes volumeMap
)

func init() {
	fs.StringVar(&flagDir, "dir", "", "directory in which to create container filesystem")
	fs.StringVar(&flagStage1, "stage1-init", "./bin/init", "path to stage1 binary")
	fs.Var(&flagVolumes, "volume", "volumes to mount into the shared container environment")
	flagVolumes = volumeMap{}
}

func main() {
	fs.Parse(os.Args[1:])
	args := fs.Args()
	if len(args) < 2 || args[0] != "run" {
		fmt.Fprintf(os.Stderr, "usage: rkt run [OPTION]... IMAGE...\n")
		os.Exit(0)
	}
	images := args[1:]
	dir := flagDir
	if dir == "" {
		log.Printf("-dir unset - using temporary directory")
		var err error
		dir, err = ioutil.TempDir("", "rkt")
		if err != nil {
			log.Fatalf("error creating temporary directory: %v", err)
		}
	}

	// - Generating the Container Unique ID (UID)
	cuid, err := types.NewUUID(genUID())
	if err != nil {
		log.Fatalf("error creating UID: %v", err)
	}

	// - Generating a Container Runtime Manifest
	cm := schema.ContainerRuntimeManifest{
		ACKind: "ContainerRuntimeManifest",
		UUID:   *cuid,
		Apps:   map[types.ACLabel]schema.App{},
	}

	v, err := types.NewSemVer(rkt.Version)
	if err != nil {
		log.Fatalf("error creating version: %v", err)
	}
	cm.ACVersion = *v

	// - Fetching the specified application TAFs
	// (for now, we just assume they are local, named by their hash, and unencrypted)
	// - Unpacking the TAFs and copying the RAF for each app into the stage2

	for _, img := range images {
		log.Println("Loading app", img)
		fh, err := os.Open(img)
		if err != nil {
			log.Fatalf("error opening app: %v", err)
		}
		gz, err := gzip.NewReader(fh)
		if err != nil {
			log.Fatalf("error reading tarball: %v", err)
		}
		ad := rkt.AppMountPath(dir, img)
		err = os.MkdirAll(ad, 0776)
		if err != nil {
			log.Fatalf("error creating app directory: %v", err)
		}
		if err := taf.ExtractTar(tar.NewReader(gz), ad); err != nil {
			log.Fatalf("error extracting TAF: %v", err)
		}
		h, err := types.NewHash(img)
		if err != nil {
			log.Fatalf("bad hash: %v", err)
		}

		// TODO(jonboulle): clarify image<->appname
		mpath := filepath.Join(ad, "manifest")
		f, err := os.Open(mpath)
		if err != nil {
			log.Fatalf("error opening app manifest: %v", err)
		}
		b, err := ioutil.ReadAll(f)
		if err != nil {
			log.Fatalf("error reading app manifest: %v", err)
		}
		var am schema.AppManifest
		if err := json.Unmarshal(b, &am); err != nil {
			log.Fatalf("error unmarshaling app manifest: %v", err)
		}

		if _, ok := cm.Apps[am.Name]; ok {
			log.Fatalf("got multiple apps by name %s", am.Name)
		}

		a := schema.App{
			ImageID:   *h,
			Isolators: am.Isolators,
			// TODO(jonboulle): reimplement once type is clear
			// Annotations: am.Annotations,
		}

		cm.Apps[am.Name] = a
	}

	var sVols []types.Volume
	for key, path := range flagVolumes {
		v := types.Volume{
			Kind:     "host",
			Source:   path,
			ReadOnly: true,
			Fulfills: []types.ACLabel{
				types.ACLabel(key),
			},
		}
		sVols = append(sVols, v)
	}
	cm.Volumes = sVols

	cdoc, err := json.Marshal(cm)
	if err != nil {
		log.Fatalf("error marshalling container manifest: %v", err)
	}

	// - Creating a filesystem for the container
	if err := os.MkdirAll(dir, 0700); err != nil {
		log.Fatalf("error creating directory: %v", err)
	}

	log.Printf("Writing container manifest")
	fn := filepath.Join(dir, "container")
	if err := ioutil.WriteFile(fn, cdoc, 0700); err != nil {
		log.Fatalf("error writing container manifest: %v", err)
	}

	log.Printf("Writing stage1 init")
	in, err := os.Open(flagStage1)
	if err != nil {
		log.Fatalf("error loading stage1 binary: %v", err)
	}
	fn = filepath.Join(dir, "stage1", "init")
	out, err := os.OpenFile(fn, os.O_CREATE|os.O_WRONLY, 0555)
	if err != nil {
		log.Fatalf("error opening stage1 init for writing: %v", err)
	}
	if _, err := io.Copy(out, in); err != nil {
		log.Fatalf("error writing stage1 init: %v", err)
	}
	if err := out.Close(); err != nil {
		log.Fatalf("error closing stage1 init: %v", err)
	}

	log.Printf("Wrote filesystem to %s\n", dir)

	log.Printf("Pivoting to filesystem")
	if err := os.Chdir(dir); err != nil {
		log.Fatalf("failed changing to dir: %v", err)
	}
	log.Printf("Execing stage1/init")
	if err := syscall.Exec("stage1/init", []string{"stage1/init"}, os.Environ()); err != nil {
		log.Fatalf("error execing init: %v", err)
	}
}

// newSchemaApp creates a new schema.App from a command-line name
func newSchemaApp(name types.ACLabel, iid types.Hash) (*schema.App, error) {
	// TODO(jonboulle): implement me properly
	a := schema.App{
		ImageID:     iid,
		Isolators:   nil,
		Annotations: nil,
	}
	return &a, nil
}

// genUID generates a unique ID for the container
// TODO(jonboulle): implement me properly - how is this generated?
func genUID() string {
	return "6733C088-A507-4694-AABF-EDBE4FC5266F"
}

// volumeMap implements the flag.Value interface to contain a set of mappings
// from mount label --> mount path
type volumeMap map[string]string

func (vm *volumeMap) Set(s string) error {
	elems := strings.Split(s, ":")
	if len(elems) != 2 {
		return errors.New("volume must be of form key:path")
	}
	key := elems[0]
	if _, ok := (*vm)[key]; ok {
		return fmt.Errorf("got multiple flags for volume %q", key)
	}
	(*vm)[key] = elems[1]
	return nil
}

func (vm *volumeMap) String() string {
	var ss []string
	for k, v := range *vm {
		ss = append(ss, fmt.Sprintf("%s:%s", k, v))
	}
	return strings.Join(ss, ",")
}
