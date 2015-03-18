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

package docker2aci

import (
	"fmt"
	"path"
	"strings"
)

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

func quote(l []string) []string {
	var quoted []string

	for _, s := range l {
		quoted = append(quoted, fmt.Sprintf("%q", s))
	}

	return quoted
}
