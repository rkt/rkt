package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
)

var defaultConfigDir = "/etc/rkt/net.d"

// NetworkConfig represents an ipmanager network configuration.
type NetworkConfig struct {
	DomainName        string   `json:"domain-name"`
	DomainNameServers []string `json:"domain-name-servers"`
	Name              string   `json:"name"`
	RangeStart        string   `json:"range-start"`
	RangeEnd          string   `json:"range-end"`
	Routers           []string `json:"routers"`
	Subnet            string   `json:"subnet"`
}

// NewNetworkConfig creates a NetworkConfig from the given network name.
func NewNetworkConfig(name string) (*NetworkConfig, error) {
	p := filepath.Join(defaultConfigDir, fmt.Sprintf("%s.conf", name))
	var c NetworkConfig
	data, err := ioutil.ReadFile(p)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(data, &c)
	return &c, err
}
