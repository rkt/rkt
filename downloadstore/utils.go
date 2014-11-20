package main

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net/url"
)

// copy the default of git which is a two byte prefix. We will likely want to
// add re-sharding later.
func blockTransform(s string) []string {
	pathSlice := make([]string, 1)
	pathSlice[0] = s[0:2]
	return pathSlice
}

func sha256sum(s string) string {
	h := sha256.New()
	io.WriteString(h, s)
	return fmt.Sprintf("%x", h.Sum(nil))
}

func parseAlways(s string) *url.URL {
	u, _ := url.Parse(s)
	return u
}
