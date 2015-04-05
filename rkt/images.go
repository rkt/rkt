// Copyright 2015 CoreOS, Inc.
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

//+build linux

package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	docker2aci "github.com/coreos/rkt/Godeps/_workspace/src/github.com/appc/docker2aci/lib"
	"github.com/coreos/rkt/Godeps/_workspace/src/github.com/appc/spec/aci"
	"github.com/coreos/rkt/Godeps/_workspace/src/github.com/appc/spec/discovery"
	"github.com/coreos/rkt/Godeps/_workspace/src/github.com/appc/spec/schema/types"
	"github.com/coreos/rkt/Godeps/_workspace/src/github.com/mitchellh/ioprogress"
	"github.com/coreos/rkt/Godeps/_workspace/src/golang.org/x/crypto/openpgp"
	"github.com/coreos/rkt/cas"
	"github.com/coreos/rkt/common/apps"
	"github.com/coreos/rkt/pkg/keystore"
)

// findImages uses findImage to attain a list of image hashes using discovery if necessary
func findImages(al *apps.Apps, ds *cas.Store, ks *keystore.Keystore) error {
	return al.Walk(func(app *apps.App) error {
		h, err := findImage(app.Image, app.Asc, ds, ks, true)
		if err != nil {
			return err
		}
		app.ImageID = *h
		return nil
	})
}

// findImage will recognize a ACI hash and use that, import a local file, use
// discovery or download an ACI directly.
func findImage(img string, asc string, ds *cas.Store, ks *keystore.Keystore, discover bool) (*types.Hash, error) {
	// check if it is a valid hash, if so let it pass through
	h, err := types.NewHash(img)
	if err == nil {
		fullKey, err := ds.ResolveKey(img)
		if err != nil {
			return nil, fmt.Errorf("could not resolve key: %v", err)
		}
		h, err = types.NewHash(fullKey)
		if err != nil {
			// should never happen
			panic(err)
		}
		return h, nil
	}

	// try fetching the image, potentially remotely
	key, err := fetchImage(img, asc, ds, ks, discover)
	if err != nil {
		return nil, err
	}
	h, err = types.NewHash(key)
	if err != nil {
		// should never happen
		panic(err)
	}

	return h, nil
}

// fetchImage will take an image as either a URL or a name string and import it
// into the store if found.  If discover is true meta-discovery is enabled.
// If asc is not "", it must exist as a local file and will be used as the
// signature file for verification, unless verification is disabled.
func fetchImage(img string, asc string, ds *cas.Store, ks *keystore.Keystore, discover bool) (string, error) {
	var (
		ascFile *os.File
		err     error
	)
	if asc != "" && ks != nil {
		ascFile, err = os.Open(asc)
		if err != nil {
			return "", fmt.Errorf("unable to open signature file: %v", err)
		}
		defer ascFile.Close()
	}

	u, err := url.Parse(img)
	if err != nil {
		return "", fmt.Errorf("not a valid image reference (%s)", img)
	}

	// if img refers to a local file, ensure the scheme is file:// and make the url path absolute
	_, err = os.Stat(u.Path)
	if err == nil {
		u.Path, err = filepath.Abs(u.Path)
		if err != nil {
			return "", fmt.Errorf("unable to get abs path: %v", err)
		}
		u.Scheme = "file"
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("unable to access %q: %v", img, err)
	}

	if discover && u.Scheme == "" {
		if app := newDiscoveryApp(img); app != nil {
			stdout("rkt: searching for app image %s", img)
			ep, attempts, err := discovery.DiscoverEndpoints(*app, true)

			if globalFlags.Debug {
				for _, a := range attempts {
					stderr("meta tag 'ac-discovery' not found on %s: %v", a.Prefix, a.Error)
				}
			}

			if err != nil {
				return "", err
			}

			if len(ep.ACIEndpoints) == 0 {
				return "", fmt.Errorf("no endpoints discovered")
			}

			latest := false
			// No specified version label, mark it as latest
			if _, ok := app.Labels["version"]; !ok {
				latest = true
			}
			return fetchImageFromEndpoints(ep, ascFile, ds, ks, latest)
		}
	}

	switch u.Scheme {
	case "http", "https", "docker", "file":
	default:
		return "", fmt.Errorf("rkt only supports http, https or docker URLs (%s)", img)
	}
	return fetchImageFromURL(u.String(), u.Scheme, ascFile, ds, ks, false)
}

func fetchImageFromEndpoints(ep *discovery.Endpoints, ascFile *os.File, ds *cas.Store, ks *keystore.Keystore, latest bool) (string, error) {
	return fetchImageFrom(ep.ACIEndpoints[0].ACI, ep.ACIEndpoints[0].ASC, "", ascFile, ds, ks, latest)
}

func fetchImageFromURL(imgurl string, scheme string, ascFile *os.File, ds *cas.Store, ks *keystore.Keystore, latest bool) (string, error) {
	return fetchImageFrom(imgurl, ascURLFromImgURL(imgurl), scheme, ascFile, ds, ks, latest)
}

func fetchImageFrom(aciURL string, ascURL string, scheme string, ascFile *os.File, ds *cas.Store, ks *keystore.Keystore, latest bool) (string, error) {
	if scheme != "file" || globalFlags.Debug {
		stdout("rkt: fetching image from %s", aciURL)
	}

	if globalFlags.InsecureSkipVerify {
		if ks != nil {
			stdout("rkt: warning: signature verification has been disabled")
		}
	} else if scheme == "docker" {
		return "", fmt.Errorf("signature verification for docker images is not supported (try --insecure-skip-verify)")
	}
	var key string
	rem, ok, err := ds.GetRemote(aciURL)
	if err == nil {
		key = rem.BlobKey
	} else {
		return "", err
	}
	if !ok {
		entity, aciFile, err := fetch(aciURL, ascURL, ascFile, ds, ks)
		if err != nil {
			return "", err
		}
		if scheme != "file" {
			defer os.Remove(aciFile.Name())
		}

		if entity != nil && !globalFlags.InsecureSkipVerify {
			fmt.Println("rkt: signature verified: ")
			for _, v := range entity.Identities {
				stdout("  %s", v.Name)
			}
		}
		key, err = ds.WriteACI(aciFile, latest)
		if err != nil {
			return "", err
		}

		if scheme != "file" {
			rem = cas.NewRemote(aciURL, ascURL)
			rem.BlobKey = key
			err = ds.WriteRemote(rem)
			if err != nil {
				return "", err
			}
		}
	}
	return key, nil
}

// fetch opens/downloads and verifies the remote ACI.
// If ascFile is not nil, it will be used as the signature file and ascURL will be ignored.
// If Keystore is nil signature verification will be skipped, regardless of ascFile.
// fetch returns the signer, an *os.File representing the ACI, and an error if any.
// err will be nil if the ACI fetches successfully and the ACI is verified.
func fetch(aciURL string, ascURL string, ascFile *os.File, ds *cas.Store, ks *keystore.Keystore) (*openpgp.Entity, *os.File, error) {
	var entity *openpgp.Entity
	u, err := url.Parse(aciURL)
	if err != nil {
		return nil, nil, fmt.Errorf("error parsing ACI url: %v", err)
	}
	if u.Scheme == "docker" {
		registryURL := strings.TrimPrefix(aciURL, "docker://")

		tmpDir, err := ds.TmpDir()
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

	if ks != nil && ascFile == nil {
		u, err := url.Parse(ascURL)
		if err != nil {
			return nil, nil, fmt.Errorf("error parsing ASC url: %v", err)
		}
		if u.Scheme == "file" {
			ascFile, err = os.Open(u.Path)
			if err != nil {
				return nil, nil, fmt.Errorf("error opening signature file: %v", err)
			}
		} else {
			stdout("Downloading signature from %v\n", ascURL)
			ascFile, err = ds.TmpFile()
			if err != nil {
				return nil, nil, fmt.Errorf("error setting up temporary file: %v", err)
			}
			if err = downloadSignatureFile(ascURL, ascFile); err != nil {
				return nil, nil, fmt.Errorf("error downloading the signature file: %v", err)
			}
			defer os.Remove(ascFile.Name())
		}
		defer ascFile.Close()
	}

	var aciFile *os.File
	if u.Scheme == "file" {
		aciFile, err = os.Open(u.Path)
		if err != nil {
			return nil, nil, fmt.Errorf("error opening ACI file: %v", err)
		}
	} else {
		aciFile, err = ds.TmpFile()
		if err != nil {
			return nil, aciFile, fmt.Errorf("error setting up temporary file: %v", err)
		}
		defer os.Remove(aciFile.Name())

		if err = downloadACI(aciURL, aciFile); err != nil {
			return nil, nil, fmt.Errorf("error downloading ACI: %v", err)
		}
	}

	if ks != nil {
		manifest, err := aci.ManifestFromImage(aciFile)
		if err != nil {
			return nil, aciFile, err
		}

		if _, err := aciFile.Seek(0, 0); err != nil {
			return nil, aciFile, fmt.Errorf("error seeking ACI file: %v", err)
		}
		if _, err := ascFile.Seek(0, 0); err != nil {
			return nil, aciFile, fmt.Errorf("error seeking signature file: %v", err)
		}
		if entity, err = ks.CheckSignature(manifest.Name.String(), aciFile, ascFile); err != nil {
			return nil, aciFile, err
		}
	}

	if _, err := aciFile.Seek(0, 0); err != nil {
		return nil, aciFile, fmt.Errorf("error seeking ACI file: %v", err)
	}
	return entity, aciFile, nil
}

type writeSyncer interface {
	io.Writer
	Sync() error
}

// downloadACI gets the aci specified at aciurl
func downloadACI(aciurl string, out writeSyncer) error {
	return downloadHTTP(aciurl, "ACI", out)
}

// downloadSignatureFile gets the signature specified at sigurl
func downloadSignatureFile(sigurl string, out writeSyncer) error {
	return downloadHTTP(sigurl, "signature", out)
}

// downloadHTTP retrieves url, creating a temp file using getTempFile
// file:// http:// and https:// urls supported
func downloadHTTP(url, label string, out writeSyncer) error {
	res, err := http.Get(url)
	if err != nil {
		return err
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
		return fmt.Errorf("bad HTTP status code: %d", res.StatusCode)
	}

	if _, err := io.Copy(out, reader); err != nil {
		return fmt.Errorf("error copying %s: %v", label, err)
	}

	if err := out.Sync(); err != nil {
		return fmt.Errorf("error writing %s: %v", label, err)
	}

	return nil
}

func ascURLFromImgURL(imgurl string) string {
	s := strings.TrimSuffix(imgurl, ".aci")
	return s + ".aci.asc"
}

// newDiscoveryApp creates a discovery app if the given img is an app name and
// has a URL-like structure, for example example.com/reduce-worker.
// Or it returns nil.
func newDiscoveryApp(img string) *discovery.App {
	app, err := discovery.NewAppFromString(img)
	if err != nil {
		return nil
	}
	u, err := url.Parse(app.Name.String())
	if err != nil || u.Scheme != "" {
		return nil
	}
	if _, ok := app.Labels["arch"]; !ok {
		app.Labels["arch"] = defaultArch
	}
	if _, ok := app.Labels["os"]; !ok {
		app.Labels["os"] = defaultOS
	}
	return app
}
