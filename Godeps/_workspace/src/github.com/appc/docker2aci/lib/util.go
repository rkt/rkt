package docker2aci

import (
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
