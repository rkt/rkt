package stage0

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
	"bytes"
	"compress/bzip2"
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"syscall"

	// WARNING: here be dragons
	// TODO(jonboulle): vendor this once the schema is stable
	"code.google.com/p/go-uuid/uuid"
	"github.com/containers/standard/schema"
	"github.com/containers/standard/schema/types"
	"github.com/containers/standard/taf"
	"github.com/coreos-inc/rkt/rkt"
)

const (
	initPath = "stage1/init"
)

type Config struct {
	ContainersDir string // root directory for rocket containers
	Stage1Init    string // binary to be execed as stage1
	Stage1Rootfs  string // compressed bundle containing a rootfs for stage1
	Debug         bool
	Images        []string          // application images (currently must be TAFs)
	Volumes       map[string]string // map of volumes that rocket can provide to applications
}

func init() {
	log.SetOutput(ioutil.Discard)
}

// Setup sets up a filesystem for a container based on the given config.
// The directory containing the filesystem is returned, and any error encountered.
func Setup(cfg Config) (string, error) {
	if cfg.Debug {
		log.SetOutput(os.Stderr)
	}

	// - Generating the Container Unique ID (UID)
	cuuid, err := types.NewUUID(uuid.New())
	if err != nil {
		return "", fmt.Errorf("error creating UID: %v", err)
	}

	// TODO(jonboulle): collision detection/mitigation
	// Create a directory for this container
	dir := filepath.Join(cfg.ContainersDir, cuuid.String())

	// - Creating a filesystem for the container
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("error creating directory: %v", err)
	}

	log.Printf("Unpacking stage1 rootfs")
	if err = unpackRootfs(cfg.Stage1Rootfs, rkt.Stage1RootfsPath(dir)); err != nil {
		return "", fmt.Errorf("error unpacking rootfs: %v", err)
	}

	log.Printf("Writing stage1 init")
	in, err := os.Open(cfg.Stage1Init)
	if err != nil {
		return "", fmt.Errorf("error loading stage1 binary: %v", err)
	}
	fn := rkt.Stage1InitPath(dir)
	out, err := os.OpenFile(fn, os.O_CREATE|os.O_WRONLY, 0555)
	if err != nil {
		return "", fmt.Errorf("error opening stage1 init for writing: %v", err)
	}
	if _, err := io.Copy(out, in); err != nil {
		return "", fmt.Errorf("error writing stage1 init: %v", err)
	}
	if err := out.Close(); err != nil {
		return "", fmt.Errorf("error closing stage1 init: %v", err)
	}

	log.Printf("Wrote filesystem to %s\n", dir)

	// - Generating a Container Runtime Manifest
	cm := schema.ContainerRuntimeManifest{
		ACKind: "ContainerRuntimeManifest",
		UUID:   *cuuid,
		Apps:   make(schema.AppList, 0),
	}

	v, err := types.NewSemVer(rkt.Version)
	if err != nil {
		return "", fmt.Errorf("error creating version: %v", err)
	}
	cm.ACVersion = *v

	// - Fetching the specified application TAFs
	//   (for now, we just assume they are local and named by their hash, and unencrypted)
	// - Unpacking the TAFs and copying the RAF for each app into the stage2

	// TODO(jonboulle): clarify imagehash<->appname. Right now we have to
	// unpack the entire TAF to access the manifest which contains the appname.

	for _, img := range cfg.Images {
		h, err := types.NewHash(img)
		if err != nil {
			return "", fmt.Errorf("error: bad image hash %q: %v", img, err)
		}
		am, err := setupImage(img, *h, dir)
		if err != nil {
			return "", fmt.Errorf("error setting up image %s: %v", img, err)
		}
		if cm.Apps.Get(am.Name) != nil {
			return "", fmt.Errorf("error: multiple apps with name %s", am.Name)
		}
		a := schema.App{
			Name:        am.Name,
			ImageID:     *h,
			Isolators:   am.Isolators,
			Annotations: am.Annotations,
		}
		cm.Apps = append(cm.Apps, a)
	}

	var sVols []types.Volume
	for key, path := range cfg.Volumes {
		v := types.Volume{
			Kind:     "host",
			Source:   path,
			ReadOnly: true,
			Fulfills: []types.ACName{
				types.ACName(key),
			},
		}
		sVols = append(sVols, v)
	}
	// TODO(jonboulle): check that app mountpoint expectations are
	// satisfied here, rather than waiting for stage1
	cm.Volumes = sVols

	cdoc, err := json.Marshal(cm)
	if err != nil {
		return "", fmt.Errorf("error marshalling container manifest: %v", err)
	}

	log.Printf("Writing container manifest")
	fn = rkt.ContainerManifestPath(dir)
	if err := ioutil.WriteFile(fn, cdoc, 0700); err != nil {
		return "", fmt.Errorf("error writing container manifest: %v", err)
	}
	return dir, nil
}

// Run actually runs the container by exec()ing the stage1 init inside
// the container filesystem.
func Run(dir string, debug bool) {
	log.Printf("Pivoting to filesystem %s", dir)
	if err := os.Chdir(dir); err != nil {
		log.Fatalf("failed changing to dir: %v", err)
	}

	log.Printf("Execing %s", initPath)
	args := []string{initPath}
	if debug {
		args = append(args, "debug")
	}
	if err := syscall.Exec(initPath, args, os.Environ()); err != nil {
		log.Fatalf("error execing init: %v", err)
	}
}

// unpackRootfs unpacks a stage1 rootfs (compressed file, pointed to by rfs)
// into dir, returning any error encountered
func unpackRootfs(rfs string, dir string) error {
	fh, err := os.Open(rfs)
	if err != nil {
		return fmt.Errorf("error opening stage1 rootfs: %v", err)
	}
	typ, err := taf.DetectFileType(fh)
	if err != nil {
		return fmt.Errorf("error detecting image type: %v", err)
	}
	if _, err := fh.Seek(0, 0); err != nil {
		return fmt.Errorf("error seeking image: %v", err)
	}
	var r io.Reader
	switch typ {
	case taf.TypeGzip:
		r, err = gzip.NewReader(fh)
		if err != nil {
			return fmt.Errorf("error reading gzip: %v", err)
		}
	case taf.TypeBzip2:
		r = bzip2.NewReader(fh)
	case taf.TypeXz:
		r = taf.XzReader(fh)
	case taf.TypeUnknown:
		return fmt.Errorf("error: unknown image filetype")
	default:
		// should never happen
		panic("no type returned from DetectFileType?")
	}
	tr := tar.NewReader(r)
	if err = os.MkdirAll(dir, 0776); err != nil {
		return fmt.Errorf("error creating stage1 rootfs directory: %v", err)
	}
	if err := taf.ExtractTar(tr, dir); err != nil {
		return fmt.Errorf("error extracting rootfs: %v", err)
	}
	return nil
}

// setupImage attempts to load the image by the given name (currently it just
// assumes it is a file in the current directory), verifies that the image
// matches the given hash (after decompression), and then extracts the image
// into a directory in the given dir.
// It returns the AppManifest that the image contains
func setupImage(img string, h types.Hash, dir string) (*schema.AppManifest, error) {
	log.Println("Loading image", img)
	fh, err := os.Open(img)
	if err != nil {
		return nil, fmt.Errorf("error opening image: %v", err)
	}
	typ, err := taf.DetectFileType(fh)
	if err != nil {
		return nil, fmt.Errorf("error detecting image type: %v", err)
	}
	if _, err := fh.Seek(0, 0); err != nil {
		return nil, fmt.Errorf("error seeking image: %v", err)
	}
	var r io.Reader
	switch typ {
	case taf.TypeGzip:
		r, err = gzip.NewReader(fh)
		if err != nil {
			return nil, fmt.Errorf("error reading gzip: %v", err)
		}
	case taf.TypeBzip2:
		r = bzip2.NewReader(fh)
	case taf.TypeXz:
		r = taf.XzReader(fh)
	case taf.TypeUnknown:
		return nil, fmt.Errorf("error: unknown image filetype")
	default:
		// should never happen
		panic("no type returned from DetectFileType?")
	}

	// Sanity check: provided image name matches image ID
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("error reading tarball: %v", err)
	}
	sum := sha256.Sum256(b)
	if id := fmt.Sprintf("%x", sum); id != h.Val {
		return nil, fmt.Errorf("image hash does not match expected")
	}

	ad := rkt.AppImagePath(dir, h)
	err = os.MkdirAll(ad, 0776)
	if err != nil {
		return nil, fmt.Errorf("error creating image directory: %v", err)
	}
	if err := taf.ExtractTar(tar.NewReader(bytes.NewReader(b)), ad); err != nil {
		return nil, fmt.Errorf("error extracting TAF: %v", err)
	}

	err = os.MkdirAll(filepath.Join(ad, "rootfs/tmp"), 0777)
	if err != nil {
		return nil, fmt.Errorf("error creating tmp directory: %v", err)
	}

	mpath := rkt.AppManifestPath(dir, h)
	f, err := os.Open(mpath)
	if err != nil {
		return nil, fmt.Errorf("error opening app manifest: %v", err)
	}
	b, err = ioutil.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("error reading app manifest: %v", err)
	}
	var am schema.AppManifest
	if err := json.Unmarshal(b, &am); err != nil {
		return nil, fmt.Errorf("error unmarshaling app manifest: %v", err)
	}
	return &am, nil
}
