// Copyright 2016 The appc Authors
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

// Package common provides misc types and variables.
package common

import (
	"fmt"
	"regexp"

	"github.com/appc/docker2aci/lib/internal/docker"
	"github.com/docker/distribution/reference"

	spec "github.com/opencontainers/image-spec/specs-go/v1"
)

type Compression int

const (
	NoCompression = iota
	GzipCompression
)

var (
	validId = regexp.MustCompile(`^(\w+:)?([A-Fa-f0-9]+)$`)
)

const (
	// AppcDockerOriginalName is the unmodified name this image was originally
	// referenced by for fetching, e.g. something like "nginx:tag" or
	// "quay.io/user/image:latest" This is identical in most cases to
	// 'registryurl/repository:tag' but may differ for the default Dockerhub
	// registry or if the tag was inferred as latest.
	AppcDockerOriginalName  = "appc.io/docker/originalname"
	AppcDockerRegistryURL   = "appc.io/docker/registryurl"
	AppcDockerRepository    = "appc.io/docker/repository"
	AppcDockerTag           = "appc.io/docker/tag"
	AppcDockerImageID       = "appc.io/docker/imageid"
	AppcDockerParentImageID = "appc.io/docker/parentimageid"
	AppcDockerEntrypoint    = "appc.io/docker/entrypoint"
	AppcDockerCmd           = "appc.io/docker/cmd"
	AppcDockerManifestHash  = "appc.io/docker/manifesthash"
)

const defaultTag = "latest"

// ParsedDockerURL represents a parsed Docker URL.
type ParsedDockerURL struct {
	OriginalName string
	IndexURL     string
	ImageName    string
	Tag          string
	Digest       string
}

type ErrSeveralImages struct {
	Msg    string
	Images []string
}

// InsecureConfig represents the different insecure options available
type InsecureConfig struct {
	SkipVerify bool
	AllowHTTP  bool
}

func (e *ErrSeveralImages) Error() string {
	return e.Msg
}

// ParseDockerURL takes a Docker URL and returns a ParsedDockerURL with its
// index URL, image name, and tag.
func ParseDockerURL(arg string) (*ParsedDockerURL, error) {
	r, err := reference.ParseNormalizedNamed(arg)
	if err != nil {
		return nil, err
	}

	var tag, digest string
	switch x := r.(type) {
	case reference.Canonical:
		digest = x.Digest().String()
	case reference.NamedTagged:
		tag = x.Tag()
	default:
		tag = defaultTag
	}

	indexURL, remoteName := docker.SplitReposName(reference.FamiliarName(r))

	return &ParsedDockerURL{
		OriginalName: arg,
		IndexURL:     indexURL,
		ImageName:    remoteName,
		Tag:          tag,
		Digest:       digest,
	}, nil
}

// ValidateLayerId validates a layer ID
func ValidateLayerId(id string) error {
	if ok := validId.MatchString(id); !ok {
		return fmt.Errorf("invalid layer ID %q", id)
	}
	return nil
}

/*
 * Media Type Selectors Section
 */

const (
	MediaTypeDockerV21Manifest       = "application/vnd.docker.distribution.manifest.v1+json"
	MediaTypeDockerV21SignedManifest = "application/vnd.docker.distribution.manifest.v1+prettyjws"
	MediaTypeDockerV21ManifestLayer  = "application/vnd.docker.container.image.rootfs.diff+x-gtar"

	MediaTypeDockerV22Manifest     = "application/vnd.docker.distribution.manifest.v2+json"
	MediaTypeDockerV22ManifestList = "application/vnd.docker.distribution.manifest.list.v2+json"
	MediaTypeDockerV22Config       = "application/vnd.docker.container.image.v1+json"
	MediaTypeDockerV22RootFS       = "application/vnd.docker.image.rootfs.diff.tar.gzip"

	MediaTypeOCIV1Manifest     = spec.MediaTypeImageManifest
	MediaTypeOCIV1ManifestList = spec.MediaTypeImageManifestList
	MediaTypeOCIV1Config       = spec.MediaTypeImageConfig
	MediaTypeOCIV1Layer        = spec.MediaTypeImageLayer
)

// MediaTypeOption represents the media types for a given docker image (or oci)
// spec.
type MediaTypeOption int

const (
	MediaTypeOptionDockerV21 = iota
	MediaTypeOptionDockerV22
	MediaTypeOptionOCIV1Pre
)

// MediaTypeSet represents a set of media types which docker2aci is to use when
// fetchimg images. As an example if a MediaTypeSet is equal to
// {MediaTypeOptionDockerV22, MediaTypeOptionOCIV1Pre}, then when an image pull
// is made V2.1 images will not be fetched. This doesn't apply to V1 pulls. As
// an edge case if a MedaTypeSet is nil or empty, that means that _every_ type
// of media type is enabled. This type is intended to be a set, and putting
// duplicates in this set is generally unadvised.
type MediaTypeSet []MediaTypeOption

func (m MediaTypeSet) ManifestMediaTypes() []string {
	if len(m) == 0 {
		return []string{
			MediaTypeDockerV21Manifest,
			MediaTypeDockerV21SignedManifest,
			MediaTypeDockerV22Manifest,
			MediaTypeOCIV1Manifest,
		}
	}
	ret := []string{}
	for _, option := range m {
		switch option {
		case MediaTypeOptionDockerV21:
			ret = append(ret, MediaTypeDockerV21Manifest)
			ret = append(ret, MediaTypeDockerV21SignedManifest)
		case MediaTypeOptionDockerV22:
			ret = append(ret, MediaTypeDockerV22Manifest)
		case MediaTypeOptionOCIV1Pre:
			ret = append(ret, MediaTypeOCIV1Manifest)
		}
	}
	return ret
}

func (m MediaTypeSet) ConfigMediaTypes() []string {
	if len(m) == 0 {
		return []string{
			MediaTypeDockerV22Config,
			MediaTypeOCIV1Config,
		}
	}
	ret := []string{}
	for _, option := range m {
		switch option {
		case MediaTypeOptionDockerV21:
		case MediaTypeOptionDockerV22:
			ret = append(ret, MediaTypeDockerV22Config)
		case MediaTypeOptionOCIV1Pre:
			ret = append(ret, MediaTypeOCIV1Config)
		}
	}
	return ret
}

func (m MediaTypeSet) LayerMediaTypes() []string {
	if len(m) == 0 {
		return []string{
			MediaTypeDockerV22RootFS,
			MediaTypeOCIV1Layer,
		}
	}
	ret := []string{}
	for _, option := range m {
		switch option {
		case MediaTypeOptionDockerV21:
		case MediaTypeOptionDockerV22:
			ret = append(ret, MediaTypeDockerV22RootFS)
		case MediaTypeOptionOCIV1Pre:
			ret = append(ret, MediaTypeOCIV1Layer)
		}
	}
	return ret
}

// RegistryOption represents a type of a registry, based on the version of the
// docker http API.
type RegistryOption int

const (
	RegistryOptionV1 = iota
	RegistryOptionV2
)

// RegistryOptionSet represents a set of registry types which docker2aci is to
// use when fetching images. As an example if a RegistryOptionSet is equal to
// {RegistryOptionV2}, then v1 pulls are disabled. As an edge case if a
// RegistryOptionSet is nil or empty, that means that _every_ type of registry
// is enabled. This type is intended to be a set, and putting duplicates in this
// set is generally unadvised.
type RegistryOptionSet []RegistryOption

func (r RegistryOptionSet) AllowsV1() bool {
	if len(r) == 0 {
		return true
	}
	for _, o := range r {
		if o == RegistryOptionV1 {
			return true
		}
	}
	return false
}

func (r RegistryOptionSet) AllowsV2() bool {
	if len(r) == 0 {
		return true
	}
	for _, o := range r {
		if o == RegistryOptionV2 {
			return true
		}
	}
	return false
}
