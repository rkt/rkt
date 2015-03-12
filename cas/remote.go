// Copyright 2014 CoreOS, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package cas implements a content-addressable-store on disk.
// It leverages the `diskv` package to store items in a simple
// key-value blob store: https://github.com/peterbourgon/diskv
package cas

import (
	"database/sql"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/coreos/rocket/pkg/keystore"

	"github.com/coreos/rocket/Godeps/_workspace/src/github.com/appc/docker2aci/lib"
	"github.com/coreos/rocket/Godeps/_workspace/src/github.com/appc/spec/aci"

	"github.com/coreos/rocket/Godeps/_workspace/src/github.com/mitchellh/ioprogress"
	"github.com/coreos/rocket/Godeps/_workspace/src/golang.org/x/crypto/openpgp"
)

func NewRemote(aciurl, sigurl string) *Remote {
	r := &Remote{
		ACIURL: aciurl,
		SigURL: sigurl,
	}
	return r
}

type Remote struct {
	ACIURL string
	SigURL string
	ETag   string
	// The key in the blob store under which the ACI has been saved.
	BlobKey string
}

// Download downloads and verifies the remote ACI.
// If Keystore is nil signature verification will be skipped.
// Download returns the signer, an *os.File representing the ACI, and an error if any.
// err will be nil if the ACI downloads successfully and the ACI is verified.
func (r Remote) Download(ds Store, ks *keystore.Keystore) (*openpgp.Entity, *os.File, error) {
	var entity *openpgp.Entity
	u, err := url.Parse(r.ACIURL)
	if err != nil {
		return nil, nil, fmt.Errorf("error parsing ACI url: %v", err)
	}
	if u.Scheme == "docker" {
		registryURL := strings.TrimPrefix(r.ACIURL, "docker://")

		tmpDir, err := ds.tmpDir()
		if err != nil {
			return nil, nil, fmt.Errorf("error creating temporary dir for docker to ACI conversion: %v", err)
		}

		acis, err := docker2aci.Convert(registryURL, true, tmpDir)
		if err != nil {
			return nil, nil, fmt.Errorf("error converting docker image to ACI: %v", err)
		}

		aciFile, err := os.Open(acis[0])
		if err != nil {
			return nil, nil, fmt.Errorf("error opening squashed ACI file: %v", err)
		}

		return nil, aciFile, nil
	}

	var sigTempFile *os.File
	if ks != nil {
		fmt.Printf("Downloading signature from %v\n", r.SigURL)
		sigTempFile, err = downloadSignatureFile(r.SigURL)
		if err != nil {
			return nil, nil, fmt.Errorf("error downloading the signature file: %v", err)
		}
		defer sigTempFile.Close()
		defer os.Remove(sigTempFile.Name())
	}

	acif, err := downloadACI(ds, r.ACIURL)
	if err != nil {
		return nil, acif, fmt.Errorf("error downloading the aci image: %v", err)
	}

	if ks != nil {
		manifest, err := aci.ManifestFromImage(acif)
		if err != nil {
			return nil, acif, err
		}

		if _, err := acif.Seek(0, 0); err != nil {
			return nil, acif, err
		}
		if _, err := sigTempFile.Seek(0, 0); err != nil {
			return nil, acif, err
		}
		if entity, err = ks.CheckSignature(manifest.Name.String(), acif, sigTempFile); err != nil {
			return nil, acif, err
		}
	}

	if _, err := acif.Seek(0, 0); err != nil {
		return nil, acif, err
	}
	return entity, acif, nil
}

// TODO: add locking
// Store stores the ACI represented by r in the target data store.
func (r Remote) Store(ds Store, aci io.Reader, latest bool) (*Remote, error) {
	key, err := ds.WriteACI(aci, latest)
	if err != nil {
		return nil, err
	}
	r.BlobKey = key
	err = ds.WriteRemote(&r)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// downloadACI gets the aci specified at aciurl
func downloadACI(ds Store, aciurl string) (*os.File, error) {
	return downloadHTTP(aciurl, "ACI", ds.tmpFile)
}

// downloadSignatureFile gets the signature specified at sigurl
func downloadSignatureFile(sigurl string) (*os.File, error) {
	getTemp := func() (*os.File, error) {
		return ioutil.TempFile("", "")
	}

	return downloadHTTP(sigurl, "signature", getTemp)
}

// downloadHTTP retrieves url, creating a temp file using getTempFile
// file:// http:// and https:// urls supported
func downloadHTTP(url, label string, getTempFile func() (*os.File, error)) (*os.File, error) {
	tmp, err := getTempFile()
	if err != nil {
		return nil, fmt.Errorf("error downloading %s: %v", label, err)
	}
	defer func() {
		if err != nil {
			os.Remove(tmp.Name())
			tmp.Close()
		}
	}()

	res, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	prefix := "Downloading " + label
	fmtBytesSize := 18
	barSize := int64(80 - len(prefix) - fmtBytesSize)
	bar := ioprogress.DrawTextFormatBar(barSize)
	fmtfunc := func(progress, total int64) string {
		return fmt.Sprintf(
			"%s: %s %s",
			prefix,
			bar(progress, total),
			ioprogress.DrawTextFormatBytes(progress, total),
		)
	}

	reader := &ioprogress.Reader{
		Reader:       res.Body,
		Size:         res.ContentLength,
		DrawFunc:     ioprogress.DrawTerminalf(os.Stdout, fmtfunc),
		DrawInterval: time.Second,
	}

	// TODO(jonboulle): handle http more robustly (redirects?)
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad HTTP status code: %d", res.StatusCode)
	}

	if _, err := io.Copy(tmp, reader); err != nil {
		return nil, fmt.Errorf("error copying %s: %v", label, err)
	}

	if err := tmp.Sync(); err != nil {
		return nil, fmt.Errorf("error writing %s: %v", label, err)
	}

	return tmp, nil
}

// GetRemote tries to retrieve a remote with the given aciURL. found will be
// false if remote doesn't exist.
func GetRemote(tx *sql.Tx, aciURL string) (remote *Remote, found bool, err error) {
	remote = &Remote{}
	rows, err := tx.Query("SELECT sigurl, etag, blobkey FROM remote WHERE aciurl == $1", aciURL)
	if err != nil {
		return nil, false, err
	}
	for rows.Next() {
		found = true
		if err := rows.Scan(&remote.SigURL, &remote.ETag, &remote.BlobKey); err != nil {
			return nil, false, err
		}
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}

	return remote, found, err
}

// WriteRemote adds or updates the provided Remote.
func WriteRemote(tx *sql.Tx, remote *Remote) error {
	// ql doesn't have an INSERT OR UPDATE function so
	// it's faster to remove and reinsert the row
	_, err := tx.Exec("DELETE FROM remote WHERE aciurl == $1", remote.ACIURL)
	if err != nil {
		return err
	}
	_, err = tx.Exec("INSERT INTO remote VALUES ($1, $2, $3, $4)", remote.ACIURL, remote.SigURL, remote.ETag, remote.BlobKey)
	if err != nil {
		return err
	}
	return nil
}
