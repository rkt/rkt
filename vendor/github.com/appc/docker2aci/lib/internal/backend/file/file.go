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

// Package file is an implementation of Docker2ACIBackend for files saved via
// "docker save".
//
// Note: this package is an implementation detail and shouldn't be used outside
// of docker2aci.
package file

import (
	"archive/tar"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/appc/docker2aci/lib/common"
	"github.com/appc/docker2aci/lib/internal"
	"github.com/appc/docker2aci/lib/internal/docker"
	"github.com/appc/docker2aci/lib/internal/tarball"
	"github.com/appc/docker2aci/lib/internal/types"
	"github.com/appc/docker2aci/lib/internal/typesV2"
	"github.com/appc/docker2aci/pkg/log"
	"github.com/appc/spec/schema"
	spec "github.com/opencontainers/image-spec/specs-go"
)

type FileBackend struct {
	file *os.File
}

func NewFileBackend(file *os.File) *FileBackend {
	return &FileBackend{
		file: file,
	}
}

func (lb *FileBackend) GetImageInfo(dockerURL string) ([]string, *types.ParsedDockerURL, error) {
	parsedDockerURL, err := docker.ParseDockerURL(dockerURL)
	if err != nil {
		// a missing Docker URL could mean that the file only contains one
		// image, so we ignore the error here, we'll handle it in getImageID
	}

	var ancestry []string
	// default file name is the tar name stripped
	name := strings.Split(filepath.Base(lb.file.Name()), ".")[0]
	appImageID, ancestry, parsedDockerURL, err := getImageID(lb.file, parsedDockerURL, name)
	if err != nil {
		return nil, nil, err
	}

	if len(ancestry) == 0 {
		ancestry, err = getAncestry(lb.file, appImageID)
		if err != nil {
			return nil, nil, fmt.Errorf("error getting ancestry: %v", err)
		}
	} else {
		// for oci the first image is the config
		ancestry = append([]string{appImageID}, ancestry...)
	}

	return ancestry, parsedDockerURL, nil
}

func (lb *FileBackend) BuildACI(layerIDs []string, dockerURL *types.ParsedDockerURL, outputDir string, tmpBaseDir string, compression common.Compression) ([]string, []*schema.ImageManifest, error) {
	if strings.Contains(layerIDs[0], ":") {
		return lb.BuildACIV22(layerIDs, dockerURL, outputDir, tmpBaseDir, compression)
	}
	var aciLayerPaths []string
	var aciManifests []*schema.ImageManifest
	var curPwl []string

	tmpDir, err := ioutil.TempDir(tmpBaseDir, "docker2aci-")
	if err != nil {
		return nil, nil, fmt.Errorf("error creating dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	for i := len(layerIDs) - 1; i >= 0; i-- {
		if err := common.ValidateLayerId(layerIDs[i]); err != nil {
			return nil, nil, err
		}
		j, err := getJson(lb.file, layerIDs[i])
		if err != nil {
			return nil, nil, fmt.Errorf("error getting layer json: %v", err)
		}

		layerData := types.DockerImageData{}
		if err := json.Unmarshal(j, &layerData); err != nil {
			return nil, nil, fmt.Errorf("error unmarshaling layer data: %v", err)
		}

		tmpLayerPath := path.Join(tmpDir, layerIDs[i])
		tmpLayerPath += ".tar"

		layerTarPath := path.Join(layerIDs[i], "layer.tar")
		layerFile, err := extractEmbeddedLayer(lb.file, layerTarPath, tmpLayerPath)
		if err != nil {
			return nil, nil, fmt.Errorf("error getting layer from file: %v", err)
		}
		defer layerFile.Close()

		log.Debug("Generating layer ACI...")
		aciPath, manifest, err := internal.GenerateACI(i, layerData, dockerURL, outputDir, layerFile, curPwl, compression)
		if err != nil {
			return nil, nil, fmt.Errorf("error generating ACI: %v", err)
		}

		aciLayerPaths = append(aciLayerPaths, aciPath)
		aciManifests = append(aciManifests, manifest)
		curPwl = manifest.PathWhitelist
	}

	return aciLayerPaths, aciManifests, nil
}

func (lb *FileBackend) BuildACIV22(layerIDs []string, dockerURL *types.ParsedDockerURL, outputDir string, tmpBaseDir string, compression common.Compression) ([]string, []*schema.ImageManifest, error) {
	if len(layerIDs) < 2 {
		return nil, nil, fmt.Errorf("insufficient layers for oci image")
	}
	var aciLayerPaths []string
	var aciManifests []*schema.ImageManifest
	var curPwl []string

	imageID := layerIDs[0]
	layerIDs = layerIDs[1:]

	j, err := getJsonV22(lb.file, imageID)
	if err != nil {
		return nil, nil, fmt.Errorf("error getting layer from file: %v", err)
	}
	imageConfig := typesV2.ImageConfig{}
	if err := json.Unmarshal(j, &imageConfig); err != nil {
		return nil, nil, fmt.Errorf("error unmarshaling image data: %v", err)
	}

	tmpDir, err := ioutil.TempDir(tmpBaseDir, "docker2aci-")
	if err != nil {
		return nil, nil, fmt.Errorf("error creating dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)
	for i := len(layerIDs) - 1; i >= 0; i-- {
		parts := strings.Split(layerIDs[i], ":")
		tmpLayerPath := path.Join(tmpDir, parts[1])
		tmpLayerPath += ".tar"
		layerTarPath := path.Join(append([]string{"blobs"}, parts...)...)
		layerFile, err := extractEmbeddedLayer(lb.file, layerTarPath, tmpLayerPath)
		if err != nil {
			return nil, nil, fmt.Errorf("error getting layer from file: %v", err)
		}
		defer layerFile.Close()
		log.Debug("Generating layer ACI...")
		var aciPath string
		var manifest *schema.ImageManifest
		if i != 0 {
			aciPath, manifest, err = internal.GenerateACI22LowerLayer(dockerURL, parts[1], outputDir, layerFile, curPwl, compression)
		} else {
			aciPath, manifest, err = internal.GenerateACI22TopLayer(dockerURL, &imageConfig, parts[1], outputDir, layerFile, curPwl, compression, aciManifests)
		}
		if err != nil {
			return nil, nil, fmt.Errorf("error generating ACI: %v", err)
		}

		aciLayerPaths = append(aciLayerPaths, aciPath)
		aciManifests = append(aciManifests, manifest)
		curPwl = manifest.PathWhitelist
	}

	return aciLayerPaths, aciManifests, nil
}

func getImageID(file *os.File, dockerURL *types.ParsedDockerURL, name string) (string, []string, *types.ParsedDockerURL, error) {
	log.Debug("getting image id...")
	type tags map[string]string
	type apps map[string]tags

	_, err := file.Seek(0, 0)
	if err != nil {
		return "", nil, nil, fmt.Errorf("error seeking file: %v", err)
	}

	tag := "latest"
	if dockerURL != nil {
		tag = dockerURL.Tag
	}

	var imageID string
	var ancestry []string
	var appName string
	reposWalker := func(t *tarball.TarFile) error {
		clean := filepath.Clean(t.Name())
		if clean == "repositories" {
			repob, err := ioutil.ReadAll(t.TarStream)
			if err != nil {
				return fmt.Errorf("error reading repositories file: %v", err)
			}

			var repositories apps
			if err := json.Unmarshal(repob, &repositories); err != nil {
				return fmt.Errorf("error unmarshaling repositories file")
			}

			if dockerURL == nil {
				n := len(repositories)
				switch {
				case n == 1:
					for key, _ := range repositories {
						appName = key
					}
				case n > 1:
					var appNames []string
					for key, _ := range repositories {
						appNames = append(appNames, key)
					}
					return &common.ErrSeveralImages{
						Msg:    "several images found",
						Images: appNames,
					}
				default:
					return fmt.Errorf("no images found")
				}
			} else {
				appName = dockerURL.ImageName
			}

			app, ok := repositories[appName]
			if !ok {
				return fmt.Errorf("app %q not found", appName)
			}

			_, ok = app[tag]
			if !ok {
				if len(app) == 1 {
					for key, _ := range app {
						tag = key
					}
				} else {
					return fmt.Errorf("tag %q not found", tag)
				}
			}

			if dockerURL == nil {
				dockerURL = &types.ParsedDockerURL{
					IndexURL:  "",
					Tag:       tag,
					ImageName: appName,
				}
			}

			imageID = string(app[tag])
		}

		if clean == "refs/"+tag {
			refb, err := ioutil.ReadAll(t.TarStream)
			if err != nil {
				return fmt.Errorf("error reading ref descriptor for tag %s: %v", tag, err)
			}

			if dockerURL == nil {
				dockerURL = &types.ParsedDockerURL{
					IndexURL:  "",
					Tag:       tag,
					ImageName: name,
				}
			}

			var ref spec.Descriptor
			if err := json.Unmarshal(refb, &ref); err != nil {
				return fmt.Errorf("error unmarshaling ref descriptor for tag %s", tag)
			}
			imageID, ancestry, err = getDataFromManifest(file, ref.Digest)
			if err != nil {
				return err
			}
			return io.EOF
		}
		return nil
	}

	tr := tar.NewReader(file)
	if err := tarball.Walk(*tr, reposWalker); err != nil && err != io.EOF {
		return "", nil, nil, err
	}

	if imageID == "" {
		return "", nil, nil, fmt.Errorf("Could not find image")
	}

	return imageID, ancestry, dockerURL, nil
}

func getDataFromManifest(file *os.File, manifestID string) (string, []string, error) {
	_, err := file.Seek(0, 0)
	if err != nil {
		return "", nil, fmt.Errorf("error seeking file: %v", err)
	}

	parts := append([]string{"blobs"}, strings.Split(manifestID, ":")...)
	jsonPath := path.Join(parts...)

	var imageID string
	var ancestry []string
	reposWalker := func(t *tarball.TarFile) error {
		clean := filepath.Clean(t.Name())
		if clean == jsonPath {

			manb, err := ioutil.ReadAll(t.TarStream)
			if err != nil {
				return fmt.Errorf("error reading image manifest: %v", err)
			}

			var manifest typesV2.ImageManifest
			if err := json.Unmarshal(manb, &manifest); err != nil {
				return fmt.Errorf("error unmarshaling image manifest")
			}
			if manifest.Config == nil {
				return fmt.Errorf("manifest does not contain a config")
			}
			imageID = manifest.Config.Digest
			// put them in reverse order
			for i := len(manifest.Layers) - 1; i >= 0; i-- {
				ancestry = append(ancestry, manifest.Layers[i].Digest)
			}
		}
		return nil
	}

	tr := tar.NewReader(file)
	if err := tarball.Walk(*tr, reposWalker); err != nil {
		return "", nil, err
	}

	return imageID, ancestry, nil
}

func getJson(file *os.File, layerID string) ([]byte, error) {
	jsonPath := path.Join(layerID, "json")
	return getTarFileBytes(file, jsonPath)
}

func getJsonV22(file *os.File, layerID string) ([]byte, error) {
	parts := append([]string{"blobs"}, strings.Split(layerID, ":")...)
	jsonPath := path.Join(parts...)
	return getTarFileBytes(file, jsonPath)
}

func getTarFileBytes(file *os.File, path string) ([]byte, error) {
	_, err := file.Seek(0, 0)
	if err != nil {
		return nil, fmt.Errorf("error seeking file: %v", err)
	}

	var fileBytes []byte
	fileWalker := func(t *tarball.TarFile) error {
		if filepath.Clean(t.Name()) == path {
			fileBytes, err = ioutil.ReadAll(t.TarStream)
			if err != nil {
				return err
			}
		}

		return nil
	}

	tr := tar.NewReader(file)
	if err := tarball.Walk(*tr, fileWalker); err != nil {
		return nil, err
	}

	if fileBytes == nil {
		return nil, fmt.Errorf("file %q not found", path)
	}

	return fileBytes, nil
}

func extractEmbeddedLayer(file *os.File, layerTarPath string, outputPath string) (*os.File, error) {
	log.Info("Extracting ", layerTarPath, "\n")
	_, err := file.Seek(0, 0)
	if err != nil {
		return nil, fmt.Errorf("error seeking file: %v", err)
	}

	var layerFile *os.File
	fileWalker := func(t *tarball.TarFile) error {
		if filepath.Clean(t.Name()) == layerTarPath {
			layerFile, err = os.Create(outputPath)
			if err != nil {
				return fmt.Errorf("error creating layer: %v", err)
			}

			_, err = io.Copy(layerFile, t.TarStream)
			if err != nil {
				return fmt.Errorf("error getting layer: %v", err)
			}
		}

		return nil
	}

	tr := tar.NewReader(file)
	if err := tarball.Walk(*tr, fileWalker); err != nil {
		return nil, err
	}

	if layerFile == nil {
		return nil, fmt.Errorf("file %q not found", layerTarPath)
	}

	return layerFile, nil
}

// getAncestry computes an image ancestry, returning an ordered list
// of dependencies starting from the topmost image to the base.
// It checks for dependency loops via duplicate detection in the image
// chain and errors out in such cases.
func getAncestry(file *os.File, imgID string) ([]string, error) {
	var ancestry []string
	deps := make(map[string]bool)

	curImgID := imgID

	var err error
	for curImgID != "" {
		if deps[curImgID] {
			return nil, fmt.Errorf("dependency loop detected at image %q", curImgID)
		}
		deps[curImgID] = true
		ancestry = append(ancestry, curImgID)
		log.Debug(fmt.Sprintf("Getting ancestry for layer %q", curImgID))
		curImgID, err = getParent(file, curImgID)
		if err != nil {
			return nil, err
		}
	}
	return ancestry, nil
}

func getParent(file *os.File, imgID string) (string, error) {
	var parent string

	_, err := file.Seek(0, 0)
	if err != nil {
		return "", fmt.Errorf("error seeking file: %v", err)
	}

	jsonPath := filepath.Join(imgID, "json")
	parentWalker := func(t *tarball.TarFile) error {
		if filepath.Clean(t.Name()) == jsonPath {
			jsonb, err := ioutil.ReadAll(t.TarStream)
			if err != nil {
				return fmt.Errorf("error reading layer json: %v", err)
			}

			var dockerData types.DockerImageData
			if err := json.Unmarshal(jsonb, &dockerData); err != nil {
				return fmt.Errorf("error unmarshaling layer data: %v", err)
			}

			parent = dockerData.Parent
		}

		return nil
	}

	tr := tar.NewReader(file)
	if err := tarball.Walk(*tr, parentWalker); err != nil {
		return "", err
	}

	log.Debug(fmt.Sprintf("Layer %q depends on layer %q", imgID, parent))
	return parent, nil
}
