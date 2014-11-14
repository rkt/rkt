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
// - Fetching the TAFs for the specified applications
// - Unpacking the TAFs and copying the RAFs for each app into the stage2 directories
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

	// WARNING: here be dragons
	// TODO(jonboulle): vendor this once the schema is stable
	"github.com/containers/standard/schema"
	"github.com/containers/standard/taf"
)

var (
	fs = flag.NewFlagSet("rkt", flag.ExitOnError)

	flagDir     string
	flagStage1  string
	flagVolumes volumeMap
)

func init() {
	fs.StringVar(&flagDir, "dir", "", "directory in which to create container filesystem")
	fs.StringVar(&flagStage1, "stage1", "./bin/stage1", "path to stage1 binary")
	fs.Var(&flagVolumes, "volume", "volumes to mount into the shared container environment")
	flagVolumes = volumeMap{}
}

func main() {
	fs.Parse(os.Args[1:])
	args := fs.Args()
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "usage: rkt run [image_hash...]\n")
		os.Exit(0)
	}
	cmd := args[0]
	switch cmd {
	case "run":
	default:
		fmt.Fprintf(os.Stderr, "rkt: unknown subcommand: %q\n", cmd)
		os.Exit(1)
	}
	fs.Parse(args[1:])

	dir := flagDir
	if dir == "" {
		log.Printf("-dir unset - using temporary directory")
		var err error
		dir, err = ioutil.TempDir("", "rkt")
		if err != nil {
			log.Fatalf("error creating temporary directory: %v", err)
		}
	}

	apps := fs.Args()

	// - Generating the Container Unique ID (UID)
	cuid, err := schema.NewUUID(genUID())
	if err != nil {
		log.Fatalf("error creating UID: %v", err)
	}

	// - Generating a Container Runtime Manifest
	cm := &schema.ContainerManifest{
		ACType: "ContainerManifest",
		UID:    *cuid,
	}

	v, err := schema.NewSemVer("1.0.0")
	if err != nil {
		log.Fatalf("error creating version: %v", err)
	}
	cm.ACVersion = *v

	sApps := make(map[string]schema.App)
	for _, name := range apps {
		a, err := newSchemaApp(name)
		if err != nil {
			log.Fatalf("error creating app: %v", err)
		}
		sApps[name] = *a
	}
	cm.Apps = sApps

	sVols := make(map[string]schema.Volume)
	for key, path := range flagVolumes {
		sVols[key] = schema.Volume{path}
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

	// Write the container document into the filesystem
	fn := filepath.Join(dir, "container")
	if err := ioutil.WriteFile(fn, cdoc, 0700); err != nil {
		log.Fatalf("error writing container manifest: %v", err)
	}

	// - Setting up stage 1 and stage 2 directories in the filesystem
	fn = filepath.Join(dir, "stage1", "opt", "stage2")
	if err := os.MkdirAll(fn, 0700); err != nil {
		log.Fatalf("error setting up stage1: %v", err)
	}

	// - Copying the stage1 binary into the container filesystem
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

	// - Fetching the TAFs for the specified applications
	// (for now, we just assume they are local, named by their hash, and unencrypted)

	// - Unpacking the TAFs and copying the RAFs for each app into the stage2 directories
	for _, name := range apps {
		fh, err := os.Open(name)
		if err != nil {
			log.Fatalf("error opening app: %v", err)
		}
		gz, err := gzip.NewReader(fh)
		if err != nil {
			log.Fatalf("error reading tarball: %v", err)
		}
		d := filepath.Join(dir, "stage1", "opt", "stage2", name)
		err = os.MkdirAll(d, 0776)
		if err != nil {
			log.Fatalf("error creating app directory: %v", err)
		}
		if err := taf.ExtractTar(tar.NewReader(gz), d); err != nil {
			log.Fatalf("error extracting TAF: %v", err)
		}
	}

	log.Printf("Wrote filesystem to %s\n", dir)

	if err := os.Chdir(dir); err != nil {
		log.Fatalf("failed changing to dir: %v", err)
	}
}

// newSchemaApp creates a new schema.App from a command-line name
func newSchemaApp(name string) (*schema.App, error) {
	// TODO(jonboulle): implement me properly
	a := schema.App{
		ID:          name,
		Isolators:   nil,
		Annotations: nil,
		Before:      nil,
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
