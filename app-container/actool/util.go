package main

import (
	"fmt"
	"net/url"
	"runtime"
	"strings"
)

const (
	defaultVersion = "latest"
	defaultOS      = runtime.GOOS
	defaultArch    = runtime.GOARCH
)

// appFromString takes a command line app parameter and returns a map of labels.
//
// Example app parameters:
// 	example.com/reduce-worker:1.0.0
// 	example.com/reduce-worker,channel=alpha,label=value
func appFromString(app string) (out map[string]string, err error) {
	out = make(map[string]string, 0)
	app = strings.Replace(app, ":", ",ver=", -1)
	app = "name=" + app
	v, err := url.ParseQuery(strings.Replace(app, ",", "&", -1))
	if err != nil {
		return nil, err
	}
	for key, val := range v {
		if len(val) > 1 {
			return nil, fmt.Errorf("label %s with multiple values %q", key, val)
		}
		out[key] = val[0]
	}
	if out["ver"] == "" {
		out["ver"] = defaultVersion
	}
	if out["os"] == "" {
		out["os"] = defaultOS
	}
	if out["arch"] == "" {
		out["arch"] = defaultArch
	}
	return
}
