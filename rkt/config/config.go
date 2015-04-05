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

package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type Headerer interface {
	Header() http.Header
}

type Config struct {
	AuthPerHost map[string]Headerer
}

type configParser interface {
	parse(config *Config, raw []byte) error
}

var (
	parsersForKind = make(map[string]map[string]configParser)
)

func addParser(kind, version string, parser configParser) {
	if _, err := getParser(kind, version); err == nil {
		panic(fmt.Sprintf("A parser for kind %q and version %q already exist", kind, version))
	}
	if _, ok := parsersForKind[kind]; !ok {
		parsersForKind[kind] = make(map[string]configParser)
	}
	parsersForKind[kind][version] = parser
}

func GetConfig() (*Config, error) {
	configDirs := []string{
		"/etc/rkt",
		"/usr/lib/rkt",
	}
	cfg := newConfig()
	for _, cd := range configDirs {
		subcfg := newConfig()
		if valid, err := validDir(cd); err != nil {
			return nil, err
		} else if !valid {
			continue
		}
		if err := readConfigDir(subcfg, cd); err != nil {
			return nil, err
		}
		mergeConfigs(cfg, subcfg)
	}
	return cfg, nil
}

func newConfig() *Config {
	return &Config{
		AuthPerHost: make(map[string]Headerer),
	}
}

func readConfigDir(config *Config, dir string) error {
	configSubdirs := map[string][]string{
		"auth.d": []string{"auth"},
	}
	for csd, kinds := range configSubdirs {
		d := filepath.Join(dir, csd)
		if valid, err := validDir(d); err != nil {
			return err
		} else if !valid {
			continue
		}
		configWalker := getConfigWalker(config, kinds, d)
		if err := filepath.Walk(d, configWalker); err != nil {
			return err
		}
	}
	return nil
}

func validDir(path string) (bool, error) {
	fi, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	if !fi.IsDir() {
		return false, fmt.Errorf("Expected %q to be a directory", path)
	}
	return true, nil
}

func getConfigWalker(config *Config, kinds []string, root string) filepath.WalkFunc {
	return func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if path == root {
			return nil
		}
		return readFile(config, info, path, kinds)
	}
}

func readFile(config *Config, info os.FileInfo, path string, kinds []string) error {
	if valid, err := validConfigFile(info); err != nil {
		return err
	} else if !valid {
		return nil
	}
	if err := parseConfigFile(config, path, kinds); err != nil {
		return err
	}
	return nil
}

func validConfigFile(info os.FileInfo) (bool, error) {
	mode := info.Mode()
	switch {
	case mode.IsDir():
		return false, filepath.SkipDir
	case mode.IsRegular():
		return filepath.Ext(info.Name()) == ".json", nil
	case mode&os.ModeSymlink == os.ModeSymlink:
		// TODO: support symlinks?
		return false, nil
	default:
		return false, nil
	}
}

type configHeader struct {
	RktVersion string `json:"rktVersion"`
	RktKind    string `json:"rktKind"`
}

func parseConfigFile(config *Config, path string, kinds []string) error {
	raw, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	var header configHeader
	if err := json.Unmarshal(raw, &header); err != nil {
		return err
	}
	if len(header.RktKind) == 0 {
		return fmt.Errorf("No rktKind specified in %q", path)
	}
	if len(header.RktVersion) == 0 {
		return fmt.Errorf("No rktVersion specified in %q", path)
	}
	kindOk := false
	for _, kind := range kinds {
		if header.RktKind == kind {
			kindOk = true
			break
		}
	}
	if !kindOk {
		dir := filepath.Dir(path)
		base := filepath.Base(path)
		kindsStr := strings.Join(kinds, `", "`)
		return fmt.Errorf("The configuration directory %q expects to have configuration files of kinds %q, but %q has kind of %q", dir, kindsStr, base, header.RktKind)
	}
	parser, err := getParser(header.RktKind, header.RktVersion)
	if err != nil {
		return err
	}
	if err := parser.parse(config, raw); err != nil {
		return fmt.Errorf("Failed to parse %q: %v", path, err)
	}
	return nil
}

func getParser(kind, version string) (configParser, error) {
	parsers, ok := parsersForKind[kind]
	if !ok {
		return nil, fmt.Errorf("No parser available for configuration of kind %q", kind)
	}
	parser, ok := parsers[version]
	if !ok {
		return nil, fmt.Errorf("No parser available for configuration of kind %q and version %q", kind, version)
	}
	return parser, nil
}

func mergeConfigs(config *Config, subconfig *Config) {
	for host, headerer := range subconfig.AuthPerHost {
		config.AuthPerHost[host] = headerer
	}
}
