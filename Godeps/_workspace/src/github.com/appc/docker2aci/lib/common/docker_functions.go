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

package common

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"runtime"
	"strings"

	"github.com/appc/docker2aci/lib/types"
)

const (
	dockercfgFileName    = "config.json"
	dockercfgFileNameOld = ".dockercfg"
)

// Image references supported by docker2aci.
// Taken from https://github.com/docker/distribution/blob/master/reference/reference.go
//
// reference      := name [ ":" tag ]
// name           := [hostname '/'] component ['/' component]*
// hostname       := hostcomponent ['.' hostcomponent]* [':' port-number]
// hostcomponent  := /([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9-]*[a-zA-Z0-9])/
// port-number    := /[0-9]+/
// component      := alpha-numeric [separator alpha-numeric]*
// alpha-numeric  := /[a-z0-9]+/
// separator      := /[_.]|__|[-]*/
// tag            := /[\w][\w.-]{0,127}/
//
// TODO: check for digests if/when we support them
var (
	referenceRegexp = anchored(capture(nameRegexp),
		optional(literal(":"), capture(tagRegexp)))

	nameRegexp = expression(
		optional(hostnameRegexp, literal(`/`)),
		componentRegexp,
		optional(repeated(literal(`/`), componentRegexp)))

	hostnameRegexp = expression(
		hostComponentRegexp,
		optional(repeated(literal(`.`), hostComponentRegexp)),
		optional(literal(`:`), portNumberRegexp))

	hostComponentRegexp = match(`(?:[a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9-]*[a-zA-Z0-9])`)

	portNumberRegexp = match(`[0-9]+`)

	componentRegexp = expression(
		alphaNumericRegexp,
		optional(repeated(separatorRegexp, alphaNumericRegexp)))

	alphaNumericRegexp = match(`[a-z0-9]+`)

	separatorRegexp = match(`(?:[._]|__|[-]*)`)

	tagRegexp = match(`[\w][\w.-]{0,127}`)
)

// match compiles the string to a regular expression.
var match = regexp.MustCompile

// literal compiles s into a literal regular expression, escaping any regexp
// reserved characters.
func literal(s string) *regexp.Regexp {
	re := match(regexp.QuoteMeta(s))

	if _, complete := re.LiteralPrefix(); !complete {
		panic("must be a literal")
	}

	return re
}

// expression defines a full expression, where each regular expression must
// follow the previous.
func expression(res ...*regexp.Regexp) *regexp.Regexp {
	var s string
	for _, re := range res {
		s += re.String()
	}

	return match(s)
}

// optional wraps the expression in a non-capturing group and makes the
// production optional.
func optional(res ...*regexp.Regexp) *regexp.Regexp {
	return match(group(expression(res...)).String() + `?`)
}

// repeated wraps the regexp in a non-capturing group to get one or more
// matches.
func repeated(res ...*regexp.Regexp) *regexp.Regexp {
	return match(group(expression(res...)).String() + `+`)
}

// group wraps the regexp in a non-capturing group.
func group(res ...*regexp.Regexp) *regexp.Regexp {
	return match(`(?:` + expression(res...).String() + `)`)
}

// capture wraps the expression in a capturing group.
func capture(res ...*regexp.Regexp) *regexp.Regexp {
	return match(`(` + expression(res...).String() + `)`)
}

// anchored anchors the regular expression by adding start and end delimiters.
func anchored(res ...*regexp.Regexp) *regexp.Regexp {
	return match(`^` + expression(res...).String() + `$`)
}

// SplitReposName breaks a reposName into an index name and remote name
func SplitReposName(reposName string) (string, string) {
	nameParts := strings.SplitN(reposName, "/", 2)
	var indexName, remoteName string
	if len(nameParts) == 1 || (!strings.Contains(nameParts[0], ".") &&
		!strings.Contains(nameParts[0], ":") && nameParts[0] != "localhost") {
		// This is a Docker Index repos (ex: samalba/hipache or ubuntu)
		indexName = defaultIndexURL
		remoteName = reposName
	} else {
		indexName = nameParts[0]
		remoteName = nameParts[1]
	}
	return indexName, remoteName
}

// Get a repos name and returns the right reposName + tag
// The tag can be confusing because of a port in a repository name.
//     Ex: localhost.localdomain:5000/samalba/hipache:latest
func parseRepositoryTag(repos string) (string, string) {
	n := strings.LastIndex(repos, ":")
	if n < 0 {
		return repos, ""
	}
	if tag := repos[n+1:]; !strings.Contains(tag, "/") {
		return repos[:n], tag
	}
	return repos, ""
}

func decodeDockerAuth(s string) (string, string, error) {
	decoded, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return "", "", err
	}
	parts := strings.SplitN(string(decoded), ":", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid auth configuration file")
	}
	user := parts[0]
	password := strings.Trim(parts[1], "\x00")
	return user, password, nil
}

func getHomeDir() string {
	if runtime.GOOS == "windows" {
		return os.Getenv("USERPROFILE")
	}
	return os.Getenv("HOME")
}

// GetDockercfgAuth reads a ~/.dockercfg file and returns the username and password
// of the given docker index server.
func GetAuthInfo(indexServer string) (string, string, error) {
	// official docker registry
	if indexServer == defaultIndexURL {
		indexServer = defaultIndexURLAuth
	}
	dockerCfgPath := path.Join(getHomeDir(), ".docker", dockercfgFileName)
	if _, err := os.Stat(dockerCfgPath); err == nil {
		j, err := ioutil.ReadFile(dockerCfgPath)
		if err != nil {
			return "", "", err
		}
		var dockerAuth types.DockerConfigFile
		if err := json.Unmarshal(j, &dockerAuth); err != nil {
			return "", "", err
		}
		// try the normal case
		if c, ok := dockerAuth.AuthConfigs[indexServer]; ok {
			return decodeDockerAuth(c.Auth)
		}
	} else if os.IsNotExist(err) {
		oldDockerCfgPath := path.Join(getHomeDir(), dockercfgFileNameOld)
		if _, err := os.Stat(oldDockerCfgPath); err != nil {
			return "", "", nil //missing file is not an error
		}
		j, err := ioutil.ReadFile(oldDockerCfgPath)
		if err != nil {
			return "", "", err
		}
		var dockerAuthOld map[string]types.DockerAuthConfigOld
		if err := json.Unmarshal(j, &dockerAuthOld); err != nil {
			return "", "", err
		}
		if c, ok := dockerAuthOld[indexServer]; ok {
			return decodeDockerAuth(c.Auth)
		}
	} else {
		// if file is there but we can't stat it for any reason other
		// than it doesn't exist then stop
		return "", "", fmt.Errorf("%s - %v", dockerCfgPath, err)
	}
	return "", "", nil
}
