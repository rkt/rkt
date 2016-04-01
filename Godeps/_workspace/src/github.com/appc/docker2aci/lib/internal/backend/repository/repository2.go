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

package repository

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/appc/docker2aci/lib/common"
	"github.com/appc/docker2aci/lib/internal"
	"github.com/appc/docker2aci/lib/internal/types"
	"github.com/appc/docker2aci/lib/internal/util"
	"github.com/appc/docker2aci/pkg/log"
	"github.com/appc/spec/schema"
	"github.com/coreos/ioprogress"
)

const (
	defaultIndexURL = "registry-1.docker.io"
)

var validHex = regexp.MustCompile(`^([a-f0-9]{64})$`)

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

func (rb *RepositoryBackend) getImageInfoV2(dockerURL *types.ParsedDockerURL) ([]string, *types.ParsedDockerURL, error) {
	manifest, layers, err := rb.getManifestV2(dockerURL)
	if err != nil {
		return nil, nil, err
	}

	rb.imageManifests[*dockerURL] = *manifest

	return layers, dockerURL, nil
}

func (rb *RepositoryBackend) buildACIV2(layerNumber int, layerID string, dockerURL *types.ParsedDockerURL, outputDir string, tmpBaseDir string, curPwl []string, compression common.Compression) (string, *schema.ImageManifest, error) {
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

	log.Debug("Generating layer ACI...")
	aciPath, aciManifest, err := internal.GenerateACI(layerNumber, layerData, dockerURL, outputDir, layerFile, curPwl, compression)
	if err != nil {
		return "", nil, fmt.Errorf("error generating ACI: %v", err)
	}

	return aciPath, aciManifest, nil
}

func (rb *RepositoryBackend) getManifestV2(dockerURL *types.ParsedDockerURL) (*v2Manifest, []string, error) {
	url := rb.schema + path.Join(dockerURL.IndexURL, "v2", dockerURL.ImageName, "manifests", dockerURL.Tag)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, nil, err
	}

	rb.setBasicAuth(req)

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

	if err := fixManifestLayers(manifest); err != nil {
		return nil, nil, err
	}

	//TODO: verify signature here

	layers := make([]string, len(manifest.FSLayers))

	for i, layer := range manifest.FSLayers {
		layers[i] = layer.BlobSum
	}

	return manifest, layers, nil
}

func fixManifestLayers(manifest *v2Manifest) error {
	type imageV1 struct {
		ID     string
		Parent string
	}
	imgs := make([]*imageV1, len(manifest.FSLayers))
	for i := range manifest.FSLayers {
		img := &imageV1{}

		if err := json.Unmarshal([]byte(manifest.History[i].V1Compatibility), img); err != nil {
			return err
		}

		imgs[i] = img
		if err := validateV1ID(img.ID); err != nil {
			return err
		}
	}

	if imgs[len(imgs)-1].Parent != "" {
		return errors.New("Invalid parent ID in the base layer of the image.")
	}

	// check general duplicates to error instead of a deadlock
	idmap := make(map[string]struct{})

	var lastID string
	for _, img := range imgs {
		// skip IDs that appear after each other, we handle those later
		if _, exists := idmap[img.ID]; img.ID != lastID && exists {
			return fmt.Errorf("ID %+v appears multiple times in manifest", img.ID)
		}
		lastID = img.ID
		idmap[lastID] = struct{}{}
	}

	// backwards loop so that we keep the remaining indexes after removing items
	for i := len(imgs) - 2; i >= 0; i-- {
		if imgs[i].ID == imgs[i+1].ID { // repeated ID. remove and continue
			manifest.FSLayers = append(manifest.FSLayers[:i], manifest.FSLayers[i+1:]...)
			manifest.History = append(manifest.History[:i], manifest.History[i+1:]...)
		} else if imgs[i].Parent != imgs[i+1].ID {
			return fmt.Errorf("Invalid parent ID. Expected %v, got %v.", imgs[i+1].ID, imgs[i].Parent)
		}
	}

	return nil
}

func validateV1ID(id string) error {
	if ok := validHex.MatchString(id); !ok {
		return fmt.Errorf("image ID %q is invalid", id)
	}
	return nil
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
	url := rb.schema + path.Join(dockerURL.IndexURL, "v2", dockerURL.ImageName, "blobs", layerID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	rb.setBasicAuth(req)

	res, err := rb.makeRequest(req, dockerURL.ImageName)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusTemporaryRedirect || res.StatusCode == http.StatusFound {
		location := res.Header.Get("Location")
		if location != "" {
			req, err = http.NewRequest("GET", location, nil)
			if err != nil {
				return nil, err
			}
			res, err = rb.makeRequest(req, dockerURL.ImageName)
			if err != nil {
				return nil, err
			}
			defer res.Body.Close()
		}
	}

	if res.StatusCode != http.StatusOK {
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

	client := util.GetTLSClient(rb.insecure)
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
		// If the scope is empty and it shouldn't be, we can infer it based on the repo
		scope = fmt.Sprintf("repository:%s:pull", repo)
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

	rb.setBasicAuth(authReq)

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

func (rb *RepositoryBackend) setBasicAuth(req *http.Request) {
	if rb.username != "" && rb.password != "" {
		req.SetBasicAuth(rb.username, rb.password)
	}
}
