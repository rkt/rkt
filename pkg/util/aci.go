package util

import (
	"archive/tar"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/appc/spec/aci"
	"github.com/appc/spec/schema"
	"github.com/coreos/rocket/Godeps/_workspace/src/golang.org/x/crypto/openpgp"
)

// NewACI creates a new ACI with the given name.
// Used for testing.
func NewACI(name string) (*os.File, error) {
	tf, err := ioutil.TempFile("", "")
	if err != nil {
		return nil, err
	}

	manifest := fmt.Sprintf(`{"acKind":"ImageManifest","acVersion":"0.1.1","name":"%s"}`, name)

	var im schema.ImageManifest
	if err := im.UnmarshalJSON([]byte(manifest)); err != nil {
		return nil, err
	}

	tw := tar.NewWriter(tf)
	aw := aci.NewImageWriter(im, tw)
	if err := aw.Close(); err != nil {
		return nil, err
	}
	return tf, nil
}

// NewDetachedSignature creates a new openpgp armored detached signature for the given ACI
// signed with armoredPrivateKey.
func NewDetachedSignature(armoredPrivateKey string, aci io.Reader) (io.Reader, error) {
	entityList, err := openpgp.ReadArmoredKeyRing(bytes.NewBufferString(armoredPrivateKey))
	if err != nil {
		return nil, err
	}
	if len(entityList) < 1 {
		return nil, errors.New("empty entity list")
	}
	signature := &bytes.Buffer{}
	if err := openpgp.ArmoredDetachSign(signature, entityList[0], aci, nil); err != nil {
		return nil, err
	}
	return signature, nil
}
