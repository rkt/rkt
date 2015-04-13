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

	"github.com/coreos/rkt/Godeps/_workspace/src/github.com/appc/docker2aci/lib/common"
	"github.com/coreos/rkt/Godeps/_workspace/src/github.com/appc/docker2aci/lib/types"
	"github.com/coreos/rkt/Godeps/_workspace/src/github.com/appc/docker2aci/lib/util"
	"github.com/coreos/rkt/Godeps/_workspace/src/github.com/appc/spec/schema"
)

type RepoData struct {
	Tokens    []string
	Endpoints []string
	Cookie    []string
}

type RepositoryBackend struct {
	repoData *RepoData
	username string
	password string
}

func NewRepositoryBackend(username string, password string) *RepositoryBackend {
	return &RepositoryBackend{
		username: username,
		password: password,
	}
}

func (rb *RepositoryBackend) GetImageInfo(url string) ([]string, *types.ParsedDockerURL, error) {
	dockerURL := common.ParseDockerURL(url)
	repoData, err := rb.getRepoData(dockerURL.IndexURL, dockerURL.ImageName)
	if err != nil {
		return nil, nil, fmt.Errorf("error getting repository data: %v", err)
	}

	// TODO(iaguis) check more endpoints
	appImageID, err := getImageIDFromTag(repoData.Endpoints[0], dockerURL.ImageName, dockerURL.Tag, repoData)
	if err != nil {
		return nil, nil, fmt.Errorf("error getting ImageID from tag %s: %v", dockerURL.Tag, err)
	}

	ancestry, err := getAncestry(appImageID, repoData.Endpoints[0], repoData)
	if err != nil {
		return nil, nil, err
	}

	rb.repoData = repoData

	return ancestry, dockerURL, nil
}

func (rb *RepositoryBackend) BuildACI(layerID string, dockerURL *types.ParsedDockerURL, outputDir string, curPwl []string, compress bool) (string, *schema.ImageManifest, error) {
	tmpDir, err := ioutil.TempDir("", "docker2aci-")
	if err != nil {
		return "", nil, fmt.Errorf("error creating dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	j, size, err := getJson(layerID, rb.repoData.Endpoints[0], rb.repoData)
	if err != nil {
		return "", nil, fmt.Errorf("error getting image json: %v", err)
	}

	layerData := types.DockerImageData{}
	if err := json.Unmarshal(j, &layerData); err != nil {
		return "", nil, fmt.Errorf("error unmarshaling layer data: %v", err)
	}

	// remove size
	layer, err := getLayer(layerID, rb.repoData.Endpoints[0], rb.repoData, int64(size))
	if err != nil {
		return "", nil, fmt.Errorf("error getting the remote layer: %v", err)
	}
	defer layer.Close()

	layerFile, err := ioutil.TempFile(tmpDir, "dockerlayer-")
	if err != nil {
		return "", nil, fmt.Errorf("error creating layer: %v", err)
	}

	_, err = io.Copy(layerFile, layer)
	if err != nil {
		return "", nil, fmt.Errorf("error getting layer: %v", err)
	}

	layerFile.Sync()

	util.Debug("Generating layer ACI...")
	aciPath, manifest, err := common.GenerateACI(layerData, dockerURL, outputDir, layerFile, curPwl, compress)
	if err != nil {
		return "", nil, fmt.Errorf("error generating ACI: %v", err)
	}

	return aciPath, manifest, nil
}

func (rb *RepositoryBackend) getRepoData(indexURL string, remote string) (*RepoData, error) {
	client := &http.Client{}
	repositoryURL := "https://" + path.Join(indexURL, "v1", "repositories", remote, "images")

	req, err := http.NewRequest("GET", repositoryURL, nil)
	if err != nil {
		return nil, err
	}

	if rb.username != "" && rb.password != "" {
		req.SetBasicAuth(rb.username, rb.password)
	}

	req.Header.Set("X-Docker-Token", "true")

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP code: %d, URL: %s", res.StatusCode, req.URL)
	}

	var tokens []string
	if res.Header.Get("X-Docker-Token") != "" {
		tokens = res.Header["X-Docker-Token"]
	}

	var cookies []string
	if res.Header.Get("Set-Cookie") != "" {
		cookies = res.Header["Set-Cookie"]
	}

	var endpoints []string
	if res.Header.Get("X-Docker-Endpoints") != "" {
		endpoints = makeEndpointsList(res.Header["X-Docker-Endpoints"])
	} else {
		// Assume same endpoint
		endpoints = append(endpoints, indexURL)
	}

	return &RepoData{
		Endpoints: endpoints,
		Tokens:    tokens,
		Cookie:    cookies,
	}, nil
}

func getImageIDFromTag(registry string, appName string, tag string, repoData *RepoData) (string, error) {
	client := &http.Client{}
	// we get all the tags instead of directly getting the imageID of the
	// requested one (.../tags/TAG) because even though it's specified in the
	// Docker API, some registries (e.g. Google Container Registry) don't
	// implement it.
	req, err := http.NewRequest("GET", "https://"+path.Join(registry, "repositories", appName, "tags"), nil)
	if err != nil {
		return "", fmt.Errorf("failed to get Image ID: %s, URL: %s", err, req.URL)
	}

	setAuthToken(req, repoData.Tokens)
	setCookie(req, repoData.Cookie)
	res, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to get Image ID: %s, URL: %s", err, req.URL)
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return "", fmt.Errorf("HTTP code: %d. URL: %s", res.StatusCode, req.URL)
	}

	j, err := ioutil.ReadAll(res.Body)

	if err != nil {
		return "", err
	}

	var tags map[string]string

	if err := json.Unmarshal(j, &tags); err != nil {
		return "", fmt.Errorf("error unmarshaling: %v", err)
	}

	imageID, ok := tags[tag]
	if !ok {
		return "", fmt.Errorf("tag %s not found", tag)
	}

	return imageID, nil
}

func getAncestry(imgID, registry string, repoData *RepoData) ([]string, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", "https://"+path.Join(registry, "images", imgID, "ancestry"), nil)
	if err != nil {
		return nil, err
	}

	setAuthToken(req, repoData.Tokens)
	setCookie(req, repoData.Cookie)
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP code: %d. URL: %s", res.StatusCode, req.URL)
	}

	var ancestry []string

	j, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("Failed to read downloaded json: %s (%s)", err, j)
	}

	if err := json.Unmarshal(j, &ancestry); err != nil {
		return nil, fmt.Errorf("error unmarshaling: %v", err)
	}

	return ancestry, nil
}

func getJson(imgID, registry string, repoData *RepoData) ([]byte, int, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", "https://"+path.Join(registry, "images", imgID, "json"), nil)
	if err != nil {
		return nil, -1, err
	}
	setAuthToken(req, repoData.Tokens)
	setCookie(req, repoData.Cookie)
	res, err := client.Do(req)
	if err != nil {
		return nil, -1, err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return nil, -1, fmt.Errorf("HTTP code: %d, URL: %s", res.StatusCode, req.URL)
	}

	imageSize := -1

	if hdr := res.Header.Get("X-Docker-Size"); hdr != "" {
		imageSize, err = strconv.Atoi(hdr)
		if err != nil {
			return nil, -1, err
		}
	}

	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, -1, fmt.Errorf("failed to read downloaded json: %v (%s)", err, b)
	}

	return b, imageSize, nil
}

func getLayer(imgID, registry string, repoData *RepoData, imgSize int64) (io.ReadCloser, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", "https://"+path.Join(registry, "images", imgID, "layer"), nil)
	if err != nil {
		return nil, err
	}

	setAuthToken(req, repoData.Tokens)
	setCookie(req, repoData.Cookie)

	util.Info("Downloading layer: ", imgID, "\n")

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 200 {
		res.Body.Close()
		return nil, fmt.Errorf("HTTP code: %d. URL: %s", res.StatusCode, req.URL)
	}

	return res.Body, nil
}

func setAuthToken(req *http.Request, token []string) {
	if req.Header.Get("Authorization") == "" {
		req.Header.Set("Authorization", "Token "+strings.Join(token, ","))
	}
}

func setCookie(req *http.Request, cookie []string) {
	if req.Header.Get("Cookie") == "" {
		req.Header.Set("Cookie", strings.Join(cookie, ""))
	}
}

func makeEndpointsList(headers []string) []string {
	var endpoints []string

	for _, ep := range headers {
		endpointsList := strings.Split(ep, ",")
		for _, endpointEl := range endpointsList {
			endpoints = append(
				endpoints,
				// TODO(iaguis) discover if httpsOrHTTP
				path.Join(strings.TrimSpace(endpointEl), "v1"))
		}
	}

	return endpoints
}
