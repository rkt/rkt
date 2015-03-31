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

// Package docker2aci implements a simple library for converting docker images to
// App Container Images (ACIs).
package docker2aci

import (
	"archive/tar"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/coreos/rkt/Godeps/_workspace/src/github.com/appc/docker2aci/tarball"
	"github.com/coreos/rkt/Godeps/_workspace/src/github.com/appc/spec/aci"
	"github.com/coreos/rkt/Godeps/_workspace/src/github.com/appc/spec/pkg/acirenderer"
	"github.com/coreos/rkt/Godeps/_workspace/src/github.com/appc/spec/schema"
	"github.com/coreos/rkt/Godeps/_workspace/src/github.com/appc/spec/schema/types"
)

const (
	defaultTag    = "latest"
	schemaVersion = "0.1.1"
)

// Convert generates ACI images from docker registry URLs.
// It takes as input a dockerURL of the form:
//
// 	{docker registry URL}/{image name}:{tag}
//
// It then gets all the layers of the requested image and converts each of
// them to ACI.
// If the squash flag is true, it squashes all the layers in one file and
// places this file in outputDir; if it is false, it places every layer in its
// own ACI in outputDir.
// It returns the list of generated ACI paths.
func Convert(dockerURL string, squash bool, outputDir string) ([]string, error) {
	parsedURL, err := parseDockerURL(dockerURL)
	if err != nil {
		return nil, fmt.Errorf("error parsing docker url: %v\n", err)
	}

	repoData, err := getRepoData(parsedURL.IndexURL, parsedURL.ImageName)
	if err != nil {
		return nil, fmt.Errorf("error getting repository data: %v\n", err)
	}

	// TODO(iaguis) check more endpoints
	appImageID, err := getImageIDFromTag(repoData.Endpoints[0], parsedURL.ImageName, parsedURL.Tag, repoData)
	if err != nil {
		return nil, fmt.Errorf("error getting ImageID from tag %s: %v\n", parsedURL.Tag, err)
	}

	ancestry, err := getAncestry(appImageID, repoData.Endpoints[0], repoData)
	if err != nil {
		return nil, fmt.Errorf("error getting ancestry: %v\n", err)
	}

	layersOutputDir := outputDir
	if squash {
		layersOutputDir, err = ioutil.TempDir("", "docker2aci-")
		if err != nil {
			return nil, fmt.Errorf("error creating dir: %v", err)
		}
		defer os.RemoveAll(layersOutputDir)
	}

	conversionStore := NewConversionStore()

	var images acirenderer.Images
	var aciLayerPaths []string
	for i, layerID := range ancestry {
		aciPath, manifest, err := buildACI(layerID, repoData, parsedURL, layersOutputDir)
		if err != nil {
			return nil, fmt.Errorf("error building layer: %v\n", err)
		}

		key, err := conversionStore.WriteACI(aciPath)
		if err != nil {
			return nil, fmt.Errorf("error inserting in the conversion store: %v\n", err)
		}

		images = append(images, acirenderer.Image{Im: manifest, Key: key, Level: uint16(i)})
		aciLayerPaths = append(aciLayerPaths, aciPath)
	}

	if squash {
		squashedImagePath, err := SquashLayers(images, conversionStore, *parsedURL, outputDir)
		if err != nil {
			return nil, fmt.Errorf("error squashing image: %v\n", err)
		}
		aciLayerPaths = []string{squashedImagePath}
	}

	return aciLayerPaths, nil
}

func parseDockerURL(arg string) (*ParsedDockerURL, error) {
	taglessRemote, tag := parseRepositoryTag(arg)
	if tag == "" {
		tag = defaultTag
	}
	indexURL, imageName := splitReposName(taglessRemote)

	return &ParsedDockerURL{
		IndexURL:  indexURL,
		ImageName: imageName,
		Tag:       tag,
	}, nil
}

func getRepoData(indexURL string, remote string) (*RepoData, error) {
	client := &http.Client{}
	repositoryURL := "https://" + path.Join(indexURL, "v1", "repositories", remote, "images")

	req, err := http.NewRequest("GET", repositoryURL, nil)
	if err != nil {
		return nil, err
	}

	// TODO(iaguis) add auth?
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
	req, err := http.NewRequest("GET", "https://"+path.Join(registry, "repositories", appName, "tags", tag), nil)
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

	var imageID string

	if err := json.Unmarshal(j, &imageID); err != nil {
		return "", fmt.Errorf("error unmarshaling: %v", err)
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

func buildACI(layerID string, repoData *RepoData, dockerURL *ParsedDockerURL, outputDir string) (string, *schema.ImageManifest, error) {
	tmpDir, err := ioutil.TempDir("", "docker2aci-")
	if err != nil {
		return "", nil, fmt.Errorf("error creating dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	layerDest := filepath.Join(tmpDir, "layer")
	layerRootfs := filepath.Join(layerDest, "rootfs")
	err = os.MkdirAll(layerRootfs, 0700)
	if err != nil {
		return "", nil, fmt.Errorf("error creating dir: %s", layerRootfs)
	}

	j, size, err := getRemoteImageJSON(layerID, repoData.Endpoints[0], repoData)
	if err != nil {
		return "", nil, fmt.Errorf("error getting image json: %v", err)
	}

	layerData := DockerImageData{}
	if err := json.Unmarshal(j, &layerData); err != nil {
		return "", nil, fmt.Errorf("error unmarshaling layer data: %v", err)
	}

	layer, err := getRemoteLayer(layerID, repoData.Endpoints[0], repoData, int64(size))
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

	manifest, err := generateManifest(layerData, dockerURL)
	if err != nil {
		return "", nil, fmt.Errorf("error generating the manifest: %v", err)
	}

	imageName := strings.Replace(dockerURL.ImageName, "/", "-", -1)
	aciPath := imageName + "-" + layerID
	if dockerURL.Tag != "" {
		aciPath += "-" + dockerURL.Tag
	}
	if layerData.OS != "" {
		aciPath += "-" + layerData.OS
		if layerData.Architecture != "" {
			aciPath += "-" + layerData.Architecture
		}
	}
	aciPath += ".aci"

	aciPath = path.Join(outputDir, aciPath)

	if err := writeACI(layerFile, *manifest, aciPath); err != nil {
		return "", nil, fmt.Errorf("error writing ACI: %v", err)
	}

	if err := validateACI(aciPath); err != nil {
		return "", nil, fmt.Errorf("invalid aci generated: %v", err)
	}

	return aciPath, manifest, nil
}

func validateACI(aciPath string) error {
	aciFile, err := os.Open(aciPath)
	if err != nil {
		return err
	}
	defer aciFile.Close()

	reader, err := aci.NewCompressedTarReader(aciFile)
	if err != nil {
		return err
	}

	if err := aci.ValidateArchive(reader); err != nil {
		return err
	}

	return nil
}

func getRemoteImageJSON(imgID, registry string, repoData *RepoData) ([]byte, int, error) {
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

func getRemoteLayer(imgID, registry string, repoData *RepoData, imgSize int64) (io.ReadCloser, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", "https://"+path.Join(registry, "images", imgID, "layer"), nil)
	if err != nil {
		return nil, err
	}

	setAuthToken(req, repoData.Tokens)
	setCookie(req, repoData.Cookie)

	fmt.Printf("Downloading layer: %s\n", imgID)

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

func generateManifest(layerData DockerImageData, dockerURL *ParsedDockerURL) (*schema.ImageManifest, error) {
	dockerConfig := layerData.Config
	genManifest := &schema.ImageManifest{}

	appURL := dockerURL.IndexURL + "/" + dockerURL.ImageName + "-" + layerData.ID
	appURL, err := types.SanitizeACName(appURL)
	if err != nil {
		return nil, err
	}
	name, err := types.NewACName(appURL)
	if err != nil {
		return nil, err
	}
	genManifest.Name = *name

	acVersion, _ := types.NewSemVer(schemaVersion)
	genManifest.ACVersion = *acVersion

	genManifest.ACKind = types.ACKind("ImageManifest")

	var labels types.Labels
	var parentLabels types.Labels

	layer, _ := types.NewACName("layer")
	labels = append(labels, types.Label{Name: *layer, Value: layerData.ID})

	tag := dockerURL.Tag
	version, _ := types.NewACName("version")
	labels = append(labels, types.Label{Name: *version, Value: tag})

	if layerData.OS != "" {
		os, _ := types.NewACName("os")
		labels = append(labels, types.Label{Name: *os, Value: layerData.OS})
		parentLabels = append(parentLabels, types.Label{Name: *os, Value: layerData.OS})

		if layerData.Architecture != "" {
			arch, _ := types.NewACName("arch")
			parentLabels = append(parentLabels, types.Label{Name: *arch, Value: layerData.Architecture})
		}
	}

	genManifest.Labels = labels

	if dockerConfig != nil {
		exec := getExecCommand(dockerConfig.Entrypoint, dockerConfig.Cmd)
		if exec != nil {
			user, group := parseDockerUser(dockerConfig.User)
			var env types.Environment
			for _, v := range dockerConfig.Env {
				parts := strings.SplitN(v, "=", 2)
				env.Set(parts[0], parts[1])
			}
			app := &types.App{
				Exec:             exec,
				User:             user,
				Group:            group,
				Environment:      env,
				WorkingDirectory: dockerConfig.WorkingDir,
			}
			genManifest.App = app
		}
	}

	if layerData.Parent != "" {
		var dependencies types.Dependencies
		parentAppNameString := dockerURL.IndexURL + "/" + dockerURL.ImageName + "-" + layerData.Parent
		parentAppNameString, err := types.SanitizeACName(parentAppNameString)
		if err != nil {
			return nil, err
		}
		parentAppName, err := types.NewACName(parentAppNameString)
		if err != nil {
			return nil, err
		}

		dependencies = append(dependencies, types.Dependency{App: *parentAppName, Labels: parentLabels})

		genManifest.Dependencies = dependencies
	}

	return genManifest, nil
}

func getExecCommand(entrypoint []string, cmd []string) types.Exec {
	var command []string
	if entrypoint == nil && cmd == nil {
		return nil
	}
	command = append(entrypoint, cmd...)
	// non-absolute paths are not allowed, fallback to "/bin/sh -c command"
	if len(command) > 0 && !filepath.IsAbs(command[0]) {
		command_prefix := []string{"/bin/sh", "-c"}
		quoted_command := quote(command)
		command = append(command_prefix, strings.Join(quoted_command, " "))
	}
	return command
}

func parseDockerUser(dockerUser string) (string, string) {
	// if the docker user is empty assume root user and group
	if dockerUser == "" {
		return "0", "0"
	}

	dockerUserParts := strings.Split(dockerUser, ":")

	// when only the user is given, the docker spec says that the default and
	// supplementary groups of the user in /etc/passwd should be applied.
	// Assume root group for now in this case.
	if len(dockerUserParts) < 2 {
		return dockerUserParts[0], "0"
	}

	return dockerUserParts[0], dockerUserParts[1]
}

func writeACI(layer io.ReadSeeker, manifest schema.ImageManifest, output string) error {
	reader, err := aci.NewCompressedTarReader(layer)
	if err != nil {
		return err
	}

	aciFile, err := os.Create(output)
	if err != nil {
		return fmt.Errorf("error creating ACI file: %v", err)
	}
	defer aciFile.Close()

	trw := tar.NewWriter(aciFile)
	defer trw.Close()

	if err := addMinimalACIStructure(trw, manifest); err != nil {
		return fmt.Errorf("error writing rootfs entry: %v", err)
	}

	convWalker := func(t *tarball.TarFile) error {
		name := t.Name()
		if name == "./" {
			return nil
		}
		t.Header.Name = path.Join("rootfs", name)
		if strings.Contains(t.Header.Name, "/.wh.") {
			return nil
		}
		if t.Header.Typeflag == tar.TypeLink {
			t.Header.Linkname = path.Join("rootfs", t.Linkname())
		}

		if err := trw.WriteHeader(t.Header); err != nil {
			return err
		}
		if _, err := io.Copy(trw, t.TarStream); err != nil {
			return err
		}

		return nil
	}

	// Write files in rootfs/
	if err := tarball.Walk(*reader, convWalker); err != nil {
		return err
	}

	return nil
}

func addMinimalACIStructure(tarWriter *tar.Writer, manifest schema.ImageManifest) error {
	if err := writeRootfsDir(tarWriter); err != nil {
		return err
	}

	if err := writeManifest(tarWriter, manifest); err != nil {
		return err
	}

	return nil
}

func writeRootfsDir(tarWriter *tar.Writer) error {
	hdr := getGenericTarHeader()
	hdr.Name = "rootfs"
	hdr.Mode = 0755
	hdr.Size = int64(0)
	hdr.Typeflag = tar.TypeDir

	if err := tarWriter.WriteHeader(hdr); err != nil {
		return err
	}

	return nil
}

func getGenericTarHeader() *tar.Header {
	// FIXME(iaguis) Use docker image time instead of the Unix Epoch?
	hdr := &tar.Header{
		Uid:        0,
		Gid:        0,
		ModTime:    time.Unix(0, 0),
		Uname:      "0",
		Gname:      "0",
		ChangeTime: time.Unix(0, 0),
	}

	return hdr
}

func writeManifest(outputWriter *tar.Writer, manifest schema.ImageManifest) error {
	b, err := json.Marshal(manifest)
	if err != nil {
		return err
	}

	hdr := getGenericTarHeader()
	hdr.Name = "manifest"
	hdr.Mode = 0644
	hdr.Size = int64(len(b))
	hdr.Typeflag = tar.TypeReg

	if err := outputWriter.WriteHeader(hdr); err != nil {
		return err
	}
	if _, err := outputWriter.Write(b); err != nil {
		return err
	}

	return nil
}

// SquashLayers receives a list of ACI layer file names ordered from base image
// to application image and squashes them into one ACI
func SquashLayers(images []acirenderer.Image, aciRegistry acirenderer.ACIRegistry, parsedDockerURL ParsedDockerURL, outputDir string) (string, error) {
	renderedACI, err := acirenderer.GetRenderedACIFromList(images, aciRegistry)
	if err != nil {
		return "", fmt.Errorf("error rendering squashed image: %v\n", err)
	}
	manifests, err := getManifests(renderedACI, aciRegistry)
	if err != nil {
		return "", fmt.Errorf("error getting manifests: %v", err)
	}

	squashedFilename := getSquashedFilename(parsedDockerURL)
	squashedImagePath := path.Join(outputDir, squashedFilename)

	squashedImageFile, err := os.Create(squashedImagePath)
	if err != nil {
		return "", err
	}
	defer squashedImageFile.Close()

	if err := writeSquashedImage(squashedImageFile, renderedACI, aciRegistry, manifests); err != nil {
		return "", fmt.Errorf("error writing squashed image: %v", err)
	}

	if err := validateACI(squashedImagePath); err != nil {
		return "", fmt.Errorf("error validating image: %v", err)
	}

	return squashedImagePath, nil
}

func getSquashedFilename(parsedDockerURL ParsedDockerURL) string {
	squashedFilename := strings.Replace(parsedDockerURL.ImageName, "/", "-", -1)
	if parsedDockerURL.Tag != "" {
		squashedFilename += "-" + parsedDockerURL.Tag
	}
	squashedFilename += ".aci"

	return squashedFilename
}

func getManifests(renderedACI acirenderer.RenderedACI, aciRegistry acirenderer.ACIRegistry) ([]schema.ImageManifest, error) {
	var manifests []schema.ImageManifest

	for _, aci := range renderedACI {
		im, err := aciRegistry.GetImageManifest(aci.Key)
		if err != nil {
			return nil, err
		}
		manifests = append(manifests, *im)
	}

	return manifests, nil
}

func writeSquashedImage(outputFile *os.File, renderedACI acirenderer.RenderedACI, aciProvider acirenderer.ACIProvider, manifests []schema.ImageManifest) error {
	outputWriter := tar.NewWriter(outputFile)
	defer outputWriter.Close()

	for _, aciFile := range renderedACI {
		rs, err := aciProvider.ReadStream(aciFile.Key)
		if err != nil {
			return err
		}
		defer rs.Close()

		squashWalker := func(t *tarball.TarFile) error {
			cleanName := filepath.Clean(t.Name())

			if _, ok := aciFile.FileMap[cleanName]; ok {
				// we generate and add the squashed manifest later
				if cleanName == "manifest" {
					return nil
				}
				if err := outputWriter.WriteHeader(t.Header); err != nil {
					return fmt.Errorf("error writing header: %v", err)
				}
				if _, err := io.Copy(outputWriter, t.TarStream); err != nil {
					return fmt.Errorf("error copying file into the tar out: %v", err)
				}
			}
			return nil
		}

		tr := tar.NewReader(rs)
		if err := tarball.Walk(*tr, squashWalker); err != nil {
			return err
		}
	}

	if err := writeRootfsDir(outputWriter); err != nil {
		return err
	}

	finalManifest := mergeManifests(manifests)

	if err := writeManifest(outputWriter, finalManifest); err != nil {
		return err
	}

	return nil
}

func mergeManifests(manifests []schema.ImageManifest) schema.ImageManifest {
	// FIXME(iaguis) we take app layer's manifest as the final manifest for now
	manifest := manifests[0]

	manifest.Dependencies = nil

	layerIndex := -1
	for i, l := range manifest.Labels {
		if l.Name.String() == "layer" {
			layerIndex = i
		}
	}

	if layerIndex != -1 {
		manifest.Labels = append(manifest.Labels[:layerIndex], manifest.Labels[layerIndex+1:]...)
	}

	// this can't fail because the old name is legal
	nameWithoutLayerID, _ := types.NewACName(stripLayerID(manifest.Name.String()))

	manifest.Name = *nameWithoutLayerID

	return manifest
}

// striplayerID strips the layer ID from an app name:
//
// myregistry.com/organization/app-name-85738f8f9a7f1b04b5329c590ebcb9e425925c6d0984089c43a022de4f19c281
// myregistry.com/organization/app-name
func stripLayerID(layerName string) string {
	n := strings.LastIndex(layerName, "-")
	return layerName[:n]
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
