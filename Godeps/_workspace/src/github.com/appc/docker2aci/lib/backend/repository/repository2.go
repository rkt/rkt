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
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/coreos/rkt/Godeps/_workspace/src/github.com/appc/docker2aci/lib/common"
	"github.com/coreos/rkt/Godeps/_workspace/src/github.com/appc/docker2aci/lib/types"
	"github.com/coreos/rkt/Godeps/_workspace/src/github.com/appc/docker2aci/lib/util"
	"github.com/coreos/rkt/Godeps/_workspace/src/github.com/appc/spec/schema"
	"github.com/coreos/rkt/Godeps/_workspace/src/github.com/coreos/ioprogress"
)

const (
	defaultIndexURL = "registry-1.docker.io"
)

type v2Manifest struct {
	Name     string `json:"name"`
	Tag      string `json:"tag"`
	FSLayers []struct {
		BlobSum string `json:"blobSum"`
	} `json:"fsLayers"`
	History []struct {
		V1Compatibility string `json:"v1Compatibility"`
	} `json:"history"`
	Signature []byte `json:"signature"`
}

func (rb *RepositoryBackend) supportsV2(indexURL string) (bool, error) {
	url := rb.protocol() + path.Join(indexURL, "v2")

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false, err
	}

	if rb.username != "" && rb.password != "" {
		req.SetBasicAuth(rb.username, rb.password)
	}

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer res.Body.Close()

	switch res.StatusCode {
	case http.StatusOK, http.StatusUnauthorized:
		if res.Header.Get("Docker-Distribution-API-Version") == "registry/2.0" {
			return true, nil
		}
		return false, nil
	case http.StatusNotFound:
		return false, nil
	default:
		return false, fmt.Errorf("unexpected http code: %d, URL: %s", res.StatusCode, req.URL)
	}
}

func (rb *RepositoryBackend) getImageInfoV2(dockerURL *types.ParsedDockerURL) ([]string, *types.ParsedDockerURL, error) {
	manifest, layers, err := rb.getManifestV2(dockerURL)
	if err != nil {
		return nil, nil, err
	}

	rb.imageManifests[*dockerURL] = *manifest

	return layers, dockerURL, nil
}

func (rb *RepositoryBackend) buildACIV2(layerNumber int, layerID string, dockerURL *types.ParsedDockerURL, outputDir string, tmpBaseDir string, curPwl []string, compress bool) (string, *schema.ImageManifest, error) {
	manifest := rb.imageManifests[*dockerURL]

	layerIndex, err := getLayerIndex(layerID, manifest)
	if err != nil {
		return "", nil, err
	}

	if len(manifest.History) <= layerIndex {
		return "", nil, fmt.Errorf("history not found for layer %s", layerID)
	}

	layerData := types.DockerImageData{}
	if err := json.Unmarshal([]byte(manifest.History[layerIndex].V1Compatibility), &layerData); err != nil {
		return "", nil, fmt.Errorf("error unmarshaling layer data: %v", err)
	}

	tmpDir, err := ioutil.TempDir(tmpBaseDir, "docker2aci-")
	if err != nil {
		return "", nil, fmt.Errorf("error creating dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	layerFile, err := rb.getLayerV2(layerID, dockerURL, tmpDir)
	if err != nil {
		return "", nil, fmt.Errorf("error getting the remote layer: %v", err)
	}
	defer layerFile.Close()

	util.Debug("Generating layer ACI...")
	aciPath, aciManifest, err := common.GenerateACI(layerNumber, layerData, dockerURL, outputDir, layerFile, curPwl, compress)
	if err != nil {
		return "", nil, fmt.Errorf("error generating ACI: %v", err)
	}

	return aciPath, aciManifest, nil
}

func (rb *RepositoryBackend) getManifestV2(dockerURL *types.ParsedDockerURL) (*v2Manifest, []string, error) {
	url := rb.protocol() + path.Join(dockerURL.IndexURL, "v2", dockerURL.ImageName, "manifests", dockerURL.Tag)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, nil, err
	}

	if rb.username != "" && rb.password != "" {
		req.SetBasicAuth(rb.username, rb.password)
	}

	res, err := rb.makeRequest(req, dockerURL.ImageName)
	if err != nil {
		return nil, nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("unexpected http code: %d, URL: %s", res.StatusCode, req.URL)
	}

	manblob, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, nil, err
	}

	manifest := &v2Manifest{}

	err = json.Unmarshal(manblob, manifest)
	if err != nil {
		return nil, nil, err
	}

	if manifest.Name != dockerURL.ImageName {
		return nil, nil, fmt.Errorf("name doesn't match what was requested, expected: %s, downloaded: %s", dockerURL.ImageName, manifest.Name)
	}

	if manifest.Tag != dockerURL.Tag {
		return nil, nil, fmt.Errorf("tag doesn't match what was requested, expected: %s, downloaded: %s", dockerURL.Tag, manifest.Tag)
	}

	//TODO: verify signature here

	layers := make([]string, len(manifest.FSLayers))

	for i, layer := range manifest.FSLayers {
		layers[i] = layer.BlobSum
	}

	return manifest, layers, nil
}

func getLayerIndex(layerID string, manifest v2Manifest) (int, error) {
	for i, layer := range manifest.FSLayers {
		if layer.BlobSum == layerID {
			return i, nil
		}
	}
	return -1, fmt.Errorf("layer not found in manifest: %s", layerID)
}

func (rb *RepositoryBackend) getLayerV2(layerID string, dockerURL *types.ParsedDockerURL, tmpDir string) (*os.File, error) {
	url := rb.protocol() + path.Join(dockerURL.IndexURL, "v2", dockerURL.ImageName, "blobs", layerID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	res, err := rb.makeRequest(req, dockerURL.ImageName)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP code: %d. URL: %s", res.StatusCode, req.URL)
	}

	var in io.Reader
	in = res.Body

	if hdr := res.Header.Get("Content-Length"); hdr != "" {
		imgSize, err := strconv.ParseInt(hdr, 10, 64)
		if err != nil {
			return nil, err
		}

		prefix := "Downloading " + layerID[:18]
		fmtBytesSize := 18
		barSize := int64(80 - len(prefix) - fmtBytesSize)
		bar := ioprogress.DrawTextFormatBarForW(barSize, os.Stderr)
		fmtfunc := func(progress, total int64) string {
			return fmt.Sprintf(
				"%s: %s %s",
				prefix,
				bar(progress, total),
				ioprogress.DrawTextFormatBytes(progress, total),
			)
		}
		in = &ioprogress.Reader{
			Reader:       res.Body,
			Size:         imgSize,
			DrawFunc:     ioprogress.DrawTerminalf(os.Stderr, fmtfunc),
			DrawInterval: 500 * time.Millisecond,
		}
	}

	layerFile, err := ioutil.TempFile(tmpDir, "dockerlayer-")
	if err != nil {
		return nil, err
	}

	_, err = io.Copy(layerFile, in)
	if err != nil {
		return nil, err
	}

	if err := layerFile.Sync(); err != nil {
		return nil, err
	}

	return layerFile, nil
}

func (rb *RepositoryBackend) makeRequest(req *http.Request, repo string) (*http.Response, error) {
	setBearerHeader := false
	hostAuthTokens, ok := rb.hostsV2AuthTokens[req.URL.Host]
	if ok {
		authToken, ok := hostAuthTokens[repo]
		if ok {
			req.Header.Set("Authorization", "Bearer "+authToken)
			setBearerHeader = true
		}
	}

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	if res.StatusCode == http.StatusUnauthorized && setBearerHeader {
		return res, err
	}

	hdr := res.Header.Get("www-authenticate")
	if res.StatusCode != http.StatusUnauthorized || hdr == "" {
		return res, err
	}

	tokens := strings.Split(hdr, " ")
	if len(tokens) != 2 || strings.ToLower(tokens[0]) != "bearer" {
		return res, err
	}

	res.Body.Close()

	tokens = strings.Split(tokens[1], ",")

	var realm, service, scope string
	for _, token := range tokens {
		if strings.HasPrefix(token, "realm") {
			realm = strings.Trim(token[len("realm="):], "\"")
		}
		if strings.HasPrefix(token, "service") {
			service = strings.Trim(token[len("service="):], "\"")
		}
		if strings.HasPrefix(token, "scope") {
			scope = strings.Trim(token[len("scope="):], "\"")
		}
	}

	if realm == "" {
		return nil, fmt.Errorf("missing realm in bearer auth challenge")
	}
	if service == "" {
		return nil, fmt.Errorf("missing service in bearer auth challenge")
	}
	// The scope can be empty if we're not getting a token for a specific repo
	if scope == "" && repo != "" {
		return nil, fmt.Errorf("missing scope in bearer auth challenge")
	}

	authReq, err := http.NewRequest("GET", realm, nil)
	if err != nil {
		return nil, err
	}

	getParams := authReq.URL.Query()
	getParams.Add("service", service)
	if scope != "" {
		getParams.Add("scope", scope)
	}
	authReq.URL.RawQuery = getParams.Encode()

	if rb.username != "" && rb.password != "" {
		authReq.SetBasicAuth(rb.username, rb.password)
	}

	res, err = client.Do(authReq)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	switch res.StatusCode {
	case http.StatusUnauthorized:
		return nil, fmt.Errorf("unable to retrieve auth token: 401 unauthorized")
	case http.StatusOK:
		break
	default:
		return nil, fmt.Errorf("unexpected http code: %d, URL: %s", res.StatusCode, authReq.URL)
	}

	tokenBlob, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	tokenStruct := struct {
		Token string `json:"token"`
	}{}

	err = json.Unmarshal(tokenBlob, &tokenStruct)
	if err != nil {
		return nil, err
	}

	hostAuthTokens, ok = rb.hostsV2AuthTokens[req.URL.Host]
	if !ok {
		hostAuthTokens = make(map[string]string)
		rb.hostsV2AuthTokens[req.URL.Host] = hostAuthTokens
	}

	hostAuthTokens[repo] = tokenStruct.Token

	return rb.makeRequest(req, repo)
}
