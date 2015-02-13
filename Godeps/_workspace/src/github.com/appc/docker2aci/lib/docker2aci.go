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

	"github.com/coreos/rocket/Godeps/_workspace/src/github.com/appc/docker2aci/tarball"
	"github.com/coreos/rocket/Godeps/_workspace/src/github.com/appc/spec/aci"
	"github.com/coreos/rocket/Godeps/_workspace/src/github.com/appc/spec/schema"
	"github.com/coreos/rocket/Godeps/_workspace/src/github.com/appc/spec/schema/types"
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

	var aciLayerPaths []string
	for i := len(ancestry) - 1; i >= 0; i-- {
		layerID := ancestry[i]
		aciPath, err := buildACI(layerID, repoData, parsedURL, layersOutputDir)
		if err != nil {
			return nil, fmt.Errorf("error building layer: %v\n", err)
		}

		aciLayerPaths = append(aciLayerPaths, aciPath)
	}

	if squash {
		squashedFilename := strings.Replace(parsedURL.ImageName, "/", "-", -1)
		if parsedURL.Tag != "" {
			squashedFilename += "-" + parsedURL.Tag
		}
		squashedFilename += ".aci"
		squashedImagePath := path.Join(outputDir, squashedFilename)

		if err := SquashLayers(aciLayerPaths, squashedImagePath); err != nil {
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

func buildACI(layerID string, repoData *RepoData, dockerURL *ParsedDockerURL, outputDir string) (string, error) {
	tmpDir, err := ioutil.TempDir("", "docker2aci-")
	if err != nil {
		return "", fmt.Errorf("error creating dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	layerDest := filepath.Join(tmpDir, "layer")
	layerRootfs := filepath.Join(layerDest, "rootfs")
	err = os.MkdirAll(layerRootfs, 0700)
	if err != nil {
		return "", fmt.Errorf("error creating dir: %s", layerRootfs)
	}

	j, size, err := getRemoteImageJSON(layerID, repoData.Endpoints[0], repoData)
	if err != nil {
		return "", fmt.Errorf("error getting image json: %v", err)
	}

	layerData := DockerImageData{}
	if err := json.Unmarshal(j, &layerData); err != nil {
		return "", fmt.Errorf("error unmarshaling layer data: %v", err)
	}

	layer, err := getRemoteLayer(layerID, repoData.Endpoints[0], repoData, int64(size))
	if err != nil {
		return "", fmt.Errorf("error getting the remote layer: %v", err)
	}
	defer layer.Close()

	layerFile, err := ioutil.TempFile(tmpDir, "dockerlayer-")
	if err != nil {
		return "", fmt.Errorf("error creating layer: %v", err)
	}

	_, err = io.Copy(layerFile, layer)
	if err != nil {
		return "", fmt.Errorf("error getting layer: %v", err)
	}

	layerFile.Sync()

	manifest, err := generateManifest(layerData, dockerURL)
	if err != nil {
		return "", fmt.Errorf("error generating the manifest: %v", err)
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
		return "", fmt.Errorf("error writing ACI: %v", err)
	}

	if err := validateACI(aciPath); err != nil {
		return "", fmt.Errorf("invalid aci generated: %v", err)
	}

	return aciPath, nil
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
	if !filepath.IsAbs(command[0]) {
		command_prefix := []string{"/bin/sh", "-c"}
		command = append(command_prefix, strings.Join(command, " "))
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

	// Write files in rootfs/
	if err = tarball.Walk(*reader, func(t *tarball.TarFile) error {
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
	}); err != nil {
		return err
	}

	return nil
}

func addMinimalACIStructure(tarWriter *tar.Writer, manifest schema.ImageManifest) error {
	hdr := getGenericTarHeader()
	hdr.Name = "rootfs"
	hdr.Mode = 0755
	hdr.Size = int64(0)
	hdr.Typeflag = tar.TypeDir

	if err := tarWriter.WriteHeader(hdr); err != nil {
		return err
	}

	if err := writeManifest(tarWriter, manifest); err != nil {
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
func SquashLayers(layers []string, squashedImagePath string) error {
	manifests, err := getManifests(layers)
	if err != nil {
		return err
	}

	fileMap, err := getFilesToLayersMap(layers)
	if err != nil {
		return err
	}

	squashedImageFile, err := os.Create(squashedImagePath)
	if err != nil {
		return err
	}
	defer squashedImageFile.Close()

	if err := writeSquashedImage(squashedImageFile, layers, fileMap, manifests); err != nil {
		return err
	}

	if err := validateACI(squashedImagePath); err != nil {
		return err
	}

	return nil
}

func getManifests(layers []string) ([]schema.ImageManifest, error) {
	var manifests []schema.ImageManifest

	for _, aciPath := range layers {
		currentFile, err := os.Open(aciPath)
		if err != nil {
			return nil, err
		}
		defer currentFile.Close()

		manifestCur, err := aci.ManifestFromImage(currentFile)
		if err != nil {
			return nil, err
		}
		if _, err := currentFile.Seek(0, os.SEEK_SET); err != nil {
			return nil, err
		}

		manifests = append(manifests, *manifestCur)
	}

	return manifests, nil
}

func getFilesToLayersMap(layers []string) (map[string]string, error) {
	var err error
	fileMap := make(map[string]string)
	for _, aciPath := range layers {
		fileMap, err = gatherFilesToLayersMap(fileMap, aciPath)
		if err != nil {
			return nil, err
		}
	}

	return fileMap, nil
}

// gatherFilesToLayersMap accumulates a map associationg each file of the final
// image with the layer it comes from. It should be called starting from the
// base layer so that the order of files in the squashed layer is preserved
// and, if a file is present several times, the last layer is taken into
// account.
func gatherFilesToLayersMap(fileMap map[string]string, currentPath string) (map[string]string, error) {
	currentFile, err := os.Open(currentPath)
	if err != nil {
		return nil, err
	}
	defer currentFile.Close()

	reader, err := aci.NewCompressedTarReader(currentFile)
	if err != nil {
		return nil, err
	}

	if err = tarball.Walk(*reader, func(t *tarball.TarFile) error {
		if t.Name() == "manifest" {
			return nil
		}

		fileMap[t.Name()] = currentPath
		return nil
	}); err != nil {
		return nil, err
	}

	return fileMap, nil
}

func writeSquashedImage(outputFile *os.File, layers []string, fileMap map[string]string, manifests []schema.ImageManifest) error {
	outputWriter := tar.NewWriter(outputFile)
	defer outputWriter.Close()

	var err error
	for _, aciPath := range layers {
		outputWriter, err = reduceACIs(outputWriter, fileMap, aciPath)
		if err != nil {
			return err
		}
	}

	finalManifest := mergeManifests(manifests)

	if err := writeManifest(outputWriter, finalManifest); err != nil {
		return err
	}

	return nil
}

func reduceACIs(outputWriter *tar.Writer, fileMap map[string]string, currentPath string) (*tar.Writer, error) {
	currentFile, err := os.Open(currentPath)
	if err != nil {
		return nil, err
	}
	defer currentFile.Close()

	reader, err := aci.NewCompressedTarReader(currentFile)
	if err != nil {
		return nil, err
	}
	if err = tarball.Walk(*reader, func(t *tarball.TarFile) error {
		if t.Name() == "manifest" {
			return nil
		}

		if fileMap[t.Name()] == currentPath {
			if err := outputWriter.WriteHeader(t.Header); err != nil {
				return fmt.Errorf("Error writing header: %v", err)
			}
			if _, err := io.Copy(outputWriter, t.TarStream); err != nil {
				return fmt.Errorf("Error copying file into the tar out: %v", err)
			}
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return outputWriter, nil
}

func mergeManifests(manifests []schema.ImageManifest) schema.ImageManifest {
	// FIXME(iaguis) we take last layer's manifest as the final manifest for now
	manifest := manifests[len(manifests)-1]

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

	// strip layerID:
	// myregistry.com/organization/app-name-85738f8f9a7f1b04b5329c590ebcb9e425925c6d0984089c43a022de4f19c281
	// myregistry.com/organization/app-name
	n := strings.LastIndex(manifest.Name.String(), "-")
	// this can't fail because the old name is legal
	nameWithoutLayerID, _ := types.NewACName(manifest.Name.String()[:n])

	manifest.Name = *nameWithoutLayerID

	return manifest
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
