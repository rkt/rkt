package docker2aci

import (
	"fmt"
	"hash"
	"io"
	"io/ioutil"
	"os"

	"github.com/coreos/rkt/Godeps/_workspace/src/github.com/appc/spec/aci"
	"github.com/coreos/rkt/Godeps/_workspace/src/github.com/appc/spec/schema"
	"github.com/coreos/rkt/Godeps/_workspace/src/github.com/appc/spec/schema/types"
)

const (
	hashPrefix = "sha512-"
)

type aciInfo struct {
	path          string
	key           string
	ImageManifest *schema.ImageManifest
}

// ConversionStore is an simple implementation of the acirenderer.ACIRegistry
// interface. It stores the Docker layers converted to ACI so we can take
// advantage of acirenderer to generate a squashed ACI Image.
type ConversionStore struct {
	acis map[string]*aciInfo
}

func NewConversionStore() *ConversionStore {
	return &ConversionStore{acis: make(map[string]*aciInfo)}
}

func (ms *ConversionStore) WriteACI(path string) (string, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}

	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	imageID := types.NewHashSHA512(data)

	im, err := aci.ManifestFromImage(f)
	if err != nil {
		return "", err
	}

	key := imageID.String()
	ms.acis[key] = &aciInfo{path: path, key: key, ImageManifest: im}
	return key, nil
}

func (ms *ConversionStore) GetImageManifest(key string) (*schema.ImageManifest, error) {
	aci, ok := ms.acis[key]
	if !ok {
		return nil, fmt.Errorf("aci with key: %s not found", key)
	}
	return aci.ImageManifest, nil
}

func (ms *ConversionStore) GetACI(name types.ACName, labels types.Labels) (string, error) {
	for _, aci := range ms.acis {
		// we implement this function to comply with the interface so don't
		// bother implementing a proper label check
		if aci.ImageManifest.Name.String() == name.String() {
			return aci.key, nil
		}
	}
	return "", fmt.Errorf("aci not found")
}

func (ms *ConversionStore) ReadStream(key string) (io.ReadCloser, error) {
	aci, ok := ms.acis[key]
	if !ok {
		return nil, fmt.Errorf("stream for key: %s not found", key)
	}
	f, err := os.Open(aci.path)
	if err != nil {
		return nil, fmt.Errorf("error opening aci: %s", aci.path)
	}

	return f, nil
}

func (ms *ConversionStore) ResolveKey(key string) (string, error) {
	return key, nil
}

func (ms *ConversionStore) HashToKey(h hash.Hash) string {
	s := h.Sum(nil)
	return fmt.Sprintf("%s%x", hashPrefix, s)
}
