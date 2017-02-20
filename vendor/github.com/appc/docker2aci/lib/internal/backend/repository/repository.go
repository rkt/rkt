// Copyright 2015 The appc Authors
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

// Package repository is an implementation of Docker2ACIBackend for Docker
// remote registries.
//
// Note: this package is an implementation detail and shouldn't be used outside
// of docker2aci.
package repository

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/appc/docker2aci/lib/common"
	"github.com/appc/docker2aci/lib/internal/typesV2"
	"github.com/appc/docker2aci/lib/internal/util"
	"github.com/appc/docker2aci/pkg/log"
	"github.com/appc/spec/schema"
)

type registryVersion int

const (
	registryV1 registryVersion = iota
	registryV2
)

type httpStatusErr struct {
	StatusCode int
	URL        *url.URL
}

func (e httpStatusErr) Error() string {
	return fmt.Sprintf("Unexpected HTTP code: %d, URL: %s", e.StatusCode, e.URL.String())
}

func isErrHTTP404(err error) bool {
	if httperr, ok := err.(*httpStatusErr); ok && httperr.StatusCode == http.StatusNotFound {
		return true
	}
	return false
}

type RepositoryBackend struct {
	repoData          *RepoData
	username          string
	password          string
	insecure          common.InsecureConfig
	hostsV1fallback   bool
	hostsV2Support    map[string]bool
	hostsV2AuthTokens map[string]map[string]string
	schema            string
	imageManifests    map[common.ParsedDockerURL]v2Manifest
	imageV2Manifests  map[common.ParsedDockerURL]*typesV2.ImageManifest
	imageConfigs      map[common.ParsedDockerURL]*typesV2.ImageConfig
	layersIndex       map[string]int
	mediaTypes        common.MediaTypeSet
	registryOptions   common.RegistryOptionSet

	debug log.Logger
}

func NewRepositoryBackend(username, password string, insecure common.InsecureConfig, debug log.Logger, mediaTypes common.MediaTypeSet, registryOptions common.RegistryOptionSet) *RepositoryBackend {
	return &RepositoryBackend{
		username:          username,
		password:          password,
		insecure:          insecure,
		hostsV1fallback:   false,
		hostsV2Support:    make(map[string]bool),
		hostsV2AuthTokens: make(map[string]map[string]string),
		imageManifests:    make(map[common.ParsedDockerURL]v2Manifest),
		imageV2Manifests:  make(map[common.ParsedDockerURL]*typesV2.ImageManifest),
		imageConfigs:      make(map[common.ParsedDockerURL]*typesV2.ImageConfig),
		layersIndex:       make(map[string]int),
		mediaTypes:        mediaTypes,
		registryOptions:   registryOptions,
		debug:             debug,
	}
}

// GetImageInfo, given the url for a docker image, will return the
// following:
// - []string: an ordered list of all layer hashes
// - string: a unique identifier for this image, like a hash of the manifest
// - *common.ParsedDockerURL: a parsed docker URL
// - error: an error if one occurred
func (rb *RepositoryBackend) GetImageInfo(url string) ([]string, string, *common.ParsedDockerURL, error) {
	dockerURL, err := common.ParseDockerURL(url)
	if err != nil {
		return nil, "", nil, err
	}

	var supportsV2, supportsV1, ok bool
	var URLSchema string

	if supportsV2, ok = rb.hostsV2Support[dockerURL.IndexURL]; !ok {
		var err error
		URLSchema, supportsV2, err = rb.supportsRegistry(dockerURL.IndexURL, registryV2)
		if err != nil {
			return nil, "", nil, err
		}
		rb.schema = URLSchema + "://"
		rb.hostsV2Support[dockerURL.IndexURL] = supportsV2
	}

	// try v2
	if supportsV2 && rb.registryOptions.AllowsV2() {
		layers, manhash, dockerURL, err := rb.getImageInfoV2(dockerURL)
		if !isErrHTTP404(err) {
			return layers, manhash, dockerURL, err
		}
		// fallback on 404 failure
		rb.hostsV1fallback = true
		// unless we can't fallback
		if !rb.registryOptions.AllowsV1() {
			return nil, "", nil, err
		}
	}

	if !rb.registryOptions.AllowsV1() {
		return nil, "", nil, fmt.Errorf("no remaining enabled registry options")
	}

	URLSchema, supportsV1, err = rb.supportsRegistry(dockerURL.IndexURL, registryV1)
	if err != nil {
		return nil, "", nil, err
	}
	if !supportsV1 && rb.hostsV1fallback {
		return nil, "", nil, fmt.Errorf("attempted fallback to API v1 but not supported")
	}
	if !supportsV1 && !supportsV2 {
		return nil, "", nil, fmt.Errorf("registry doesn't support API v2 nor v1")
	}
	rb.schema = URLSchema + "://"
	// try v1, hard fail on failure
	return rb.getImageInfoV1(dockerURL)
}

func (rb *RepositoryBackend) BuildACI(layerIDs []string, manhash string, dockerURL *common.ParsedDockerURL, outputDir string, tmpBaseDir string, compression common.Compression) ([]string, []*schema.ImageManifest, error) {
	if rb.hostsV1fallback || !rb.hostsV2Support[dockerURL.IndexURL] {
		return rb.buildACIV1(layerIDs, manhash, dockerURL, outputDir, tmpBaseDir, compression)
	} else {
		return rb.buildACIV2(layerIDs, manhash, dockerURL, outputDir, tmpBaseDir, compression)
	}
}

// checkRegistryStatus determines registry API version compatibility according to spec:
// https://docs.docker.com/registry/spec/api/#/api-version-check
func checkRegistryStatus(statusCode int, hdr http.Header, version registryVersion) (bool, error) {
	switch statusCode {
	case http.StatusOK, http.StatusUnauthorized:
		ok := true
		if version == registryV2 {
			// According to v2 spec, registries SHOULD set this header value
			// and clients MAY fallback to v1 if missing, as done here.
			ok = hdr.Get("Docker-Distribution-API-Version") == "registry/2.0"
		}
		return ok, nil
	}
	return false, nil
}

func (rb *RepositoryBackend) supportsRegistry(indexURL string, version registryVersion) (schema string, ok bool, err error) {
	var URLPath string
	switch version {
	case registryV1:
		URLPath = "v1/_ping"
	case registryV2:
		URLPath = "v2/"
	}

	fetch := func(schema string) (res *http.Response, err error) {
		u := url.URL{Scheme: schema, Host: indexURL, Path: URLPath}
		req, err := http.NewRequest("GET", u.String(), nil)
		if err != nil {
			return nil, err
		}

		rb.setBasicAuth(req)

		client := util.GetTLSClient(rb.insecure.SkipVerify)
		res, err = client.Do(req)
		return
	}

	schema = "https"
	res, err := fetch(schema)
	if err == nil {
		ok, err = checkRegistryStatus(res.StatusCode, res.Header, version)
		defer res.Body.Close()
	}
	if err != nil || !ok {
		if rb.insecure.AllowHTTP {
			schema = "http"
			res, err = fetch(schema)
			if err == nil {
				ok, err = checkRegistryStatus(res.StatusCode, res.Header, version)
				defer res.Body.Close()
			}
		}
		return schema, ok, err
	}

	return schema, ok, err
}
