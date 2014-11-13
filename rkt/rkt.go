package main

//
// Rocket is a reference implementation of the app container specification.
//
// Execution on Rocket is divided into a number of stages, and the `rkt`
// binary implements the first stage (stage 0), which consists of the
// following tasks:
// - Generating the Container Unique ID (UID)
// - Generating the container document
// - Creating a directory for the container
// - Copying the stage1 into the container directory
// - Copying the RAFs for each app into the stage2 directories
//
// Given a run command such as:
//	rkt run --volume bind:/opt/tenant1/database \
//		example.com/data-downloader-1.0.0 \
//		example.com/ourapp-1.0.0 \
//		example.com/logbackup-1.0.0
//
// the container doc generated will be compliant with the ACE spec.
//

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	// WARNING: here be dragons
	// TODO(jonboulle): vendor this once the schema is stable
	"github.com/containers/standard/schema"
)

var (
	fs = flag.NewFlagSet("rkt", flag.ExitOnError)

	flagVolumes volumeMap
)

func init() {
	fs.Var(&flagVolumes, "volume", "volumes to mount into the shared container environment")
	flagVolumes = volumeMap{}
}

func main() {
	fs.Parse(os.Args[1:])
	args := fs.Args()
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "usage: rkt run [image...]\n")
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
	apps := fs.Args()
	cm := &schema.ContainerManifest{
		ACType: "ContainerManifest",
	}

	v, err := schema.NewSemVer("1.0.0")
	if err != nil {
		log.Fatalf("error creating version: %v", err)
	}
	cm.ACVersion = *v

	cuid, err := schema.NewUUID(genUID())
	if err != nil {
		log.Fatalf("error creating UID: %v", err)
	}
	cm.UID = *cuid

	sApps := make(map[string]schema.App)
	for _, name := range apps {
		a, err := newApp(name)
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

	b, err := json.Marshal(cm)
	if err != nil {
		log.Fatalf("error marshalling container manifest: %v", err)
	}
	fmt.Println(string(b))
}

// newApp creates a new App
// TODO(jonboulle): implement me
func newApp(name string) (*schema.App, error) {
	a := schema.App{
		ID: name,
	}
	return &a, nil
}

// genUID generates a unique ID for the container
// TODO(jonboulle): implement me
func genUID() string {
	return "6733C088-A507-4694-AABF-EDBE4FC5266F"
}

// volumeMap implements the flag.Value interface
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
