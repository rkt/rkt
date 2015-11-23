// Copyright 2015 The rkt Authors
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

package image

import (
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	"github.com/coreos/rkt/rkt/config"
	rktflag "github.com/coreos/rkt/rkt/flag"
	"github.com/coreos/rkt/store"

	docker2aci "github.com/coreos/rkt/Godeps/_workspace/src/github.com/appc/docker2aci/lib"
	d2acommon "github.com/coreos/rkt/Godeps/_workspace/src/github.com/appc/docker2aci/lib/common"
)

// dockerFetcher is used to fetch images from docker:// URLs. It uses
// a docker2aci library to perform this task.
type dockerFetcher struct {
	// TODO(krnowak): Fix the docs when we support docker image
	// verification. Will that ever happen?
	// InsecureFlags tells which insecure functionality should
	// be enabled. No image verification must be true for now.
	InsecureFlags *rktflag.SecFlags
	DockerAuth    map[string]config.BasicCredentials
	S             *store.Store
	Debug         bool
}

// GetHash uses docker2aci to download the image and convert it to
// ACI, then stores it in the store and returns the hash.
func (f *dockerFetcher) GetHash(u *url.URL) (string, error) {
	dockerURL := d2acommon.ParseDockerURL(path.Join(u.Host, u.Path))
	latest := dockerURL.Tag == "latest"
	return f.fetchImageFrom(u, latest)
}

func (f *dockerFetcher) fetchImageFrom(u *url.URL, latest bool) (string, error) {
	if !f.InsecureFlags.SkipImageCheck() {
		return "", fmt.Errorf("signature verification for docker images is not supported (try --insecure-options=image)")
	}

	if f.Debug {
		stderr("fetching image from %s", u.String())
	}

	aciFile, err := f.fetch(u)
	if err != nil {
		return "", err
	}
	// At this point, the ACI file is removed, but it is kept
	// alive, because we have an fd to it opened.
	defer aciFile.Close()

	key, err := f.S.WriteACI(aciFile, latest)
	if err != nil {
		return "", err
	}

	// TODO(krnowak): Consider dropping the signature URL part
	// from store.Remote. It is not used anywhere and the data
	// stored here is useless.
	newRem := store.NewRemote(u.String(), ascURLFromImgURL(u).String())
	newRem.BlobKey = key
	newRem.DownloadTime = time.Now()
	err = f.S.WriteRemote(newRem)
	if err != nil {
		return "", err
	}

	return key, nil
}

func (f *dockerFetcher) fetch(u *url.URL) (*os.File, error) {
	tmpDir, err := f.getTmpDir()
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)

	registryURL := strings.TrimPrefix(u.String(), "docker://")
	user, password := f.getCreds(registryURL)
	// for now, always use https:// when fetching docker images
	acis, err := docker2aci.Convert(registryURL, true /* squash */, tmpDir, tmpDir, user, password, false /* insecure */)
	if err != nil {
		return nil, fmt.Errorf("error converting docker image to ACI: %v", err)
	}

	aciFile, err := os.Open(acis[0])
	if err != nil {
		return nil, fmt.Errorf("error opening squashed ACI file: %v", err)
	}

	return aciFile, nil
}

func (f *dockerFetcher) getTmpDir() (string, error) {
	storeTmpDir, err := f.S.TmpDir()
	if err != nil {
		return "", fmt.Errorf("error creating temporary dir for docker to ACI conversion: %v", err)
	}
	return ioutil.TempDir(storeTmpDir, "docker2aci-")
}

func (f *dockerFetcher) getCreds(registryURL string) (string, string) {
	indexName := docker2aci.GetIndexName(registryURL)
	if creds, ok := f.DockerAuth[indexName]; ok {
		return creds.User, creds.Password
	}
	return "", ""
}
