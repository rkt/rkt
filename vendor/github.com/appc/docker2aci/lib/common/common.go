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
	r, err := reference.ParseNamed(arg)
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

	indexURL, remoteName := docker.SplitReposName(r.Name())

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
