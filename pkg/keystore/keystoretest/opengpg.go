// Package keystoretest provides utilities for ACI keystore testing.
//go:generate go run keygen.go
package keystoretest

import (
	"bytes"
	"errors"
	"io"

	"github.com/coreos/rocket/Godeps/_workspace/src/golang.org/x/crypto/openpgp"
)

// A KeyDetails represents an openpgp.Entity and its key details.
type KeyDetails struct {
	Fingerprint       string
	ArmoredPublicKey  string
	ArmoredPrivateKey string
}

// NewMessageAndSignature generates a new random message signed by the given entity.
// NewMessageAndSignature returns message, signature and an error if any.
func NewMessageAndSignature(armoredPrivateKey string) (io.Reader, io.Reader, error) {
	entityList, err := openpgp.ReadArmoredKeyRing(bytes.NewBufferString(armoredPrivateKey))
	if err != nil {
		return nil, nil, err
	}
	if len(entityList) < 1 {
		return nil, nil, errors.New("empty entity list")
	}
	signature := &bytes.Buffer{}
	message := []byte("data")
	if err := openpgp.ArmoredDetachSign(signature, entityList[0], bytes.NewReader(message), nil); err != nil {
		return nil, nil, err
	}
	return bytes.NewBuffer(message), signature, nil
}
