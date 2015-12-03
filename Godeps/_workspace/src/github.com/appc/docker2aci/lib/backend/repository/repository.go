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
package repository

import (
	"github.com/coreos/rkt/Godeps/_workspace/src/github.com/appc/docker2aci/lib/common"
	"github.com/coreos/rkt/Godeps/_workspace/src/github.com/appc/docker2aci/lib/types"
	"github.com/coreos/rkt/Godeps/_workspace/src/github.com/appc/spec/schema"
)

type RepositoryBackend struct {
	repoData          *RepoData
	username          string
	password          string
	insecure          bool
	hostsV2Support    map[string]bool
	hostsV2AuthTokens map[string]map[string]string
	imageManifests    map[types.ParsedDockerURL]v2Manifest
}

func NewRepositoryBackend(username string, password string, insecure bool) *RepositoryBackend {
	return &RepositoryBackend{
		username:          username,
		password:          password,
		insecure:          insecure,
		hostsV2Support:    make(map[string]bool),
		hostsV2AuthTokens: make(map[string]map[string]string),
		imageManifests:    make(map[types.ParsedDockerURL]v2Manifest),
	}
}

func (rb *RepositoryBackend) GetImageInfo(url string) ([]string, *types.ParsedDockerURL, error) {
	dockerURL := common.ParseDockerURL(url)

	var supportsV2, ok bool
	if supportsV2, ok = rb.hostsV2Support[dockerURL.IndexURL]; !ok {
		var err error
		supportsV2, err = rb.supportsV2(dockerURL.IndexURL)
		if err != nil {
			return nil, nil, err
		}
		rb.hostsV2Support[dockerURL.IndexURL] = supportsV2
	}

	if supportsV2 {
		return rb.getImageInfoV2(dockerURL)
	} else {
		return rb.getImageInfoV1(dockerURL)
	}
}

func (rb *RepositoryBackend) BuildACI(layerNumber int, layerID string, dockerURL *types.ParsedDockerURL, outputDir string, tmpBaseDir string, curPwl []string, compress bool) (string, *schema.ImageManifest, error) {
	if rb.hostsV2Support[dockerURL.IndexURL] {
		return rb.buildACIV2(layerNumber, layerID, dockerURL, outputDir, tmpBaseDir, curPwl, compress)
	} else {
		return rb.buildACIV1(layerNumber, layerID, dockerURL, outputDir, tmpBaseDir, curPwl, compress)
	}
}

func (rb *RepositoryBackend) protocol() string {
	if rb.insecure {
		return "http://"
	} else {
		return "https://"
	}
}
