package main

import (
	"fmt"

	rktnet "github.com/coreos/rkt/networking/net"
)

var defaultConfigDir = "/etc/rkt/ipam.d"

// IPAMConfig represents the IP related network configuration.
type IPAMConfig struct {
	Name       string
	Type       string   `json:"type"`
	RangeStart string   `json:"rangeStart"`
	RangeEnd   string   `json:"rangeEnd"`
	Subnet     string   `json:"subnet"`
	Gateway    string   `json:"gateway"`
	Routes     []string `json:"routes"`
}

type Net struct {
	Name string      `json:"name"`
	IPAM *IPAMConfig `json:"ipam"`
}

// NewIPAMConfig creates a NetworkConfig from the given network name.
func NewIPAMConfig(netConf string) (*IPAMConfig, error) {
	n := Net{}
	if err := rktnet.LoadNet(netConf, &n); err != nil {
		return nil, err
	}

	if n.IPAM == nil {
		return nil, fmt.Errorf("%q missing 'ipam' key")
	}

	// Copy net name into IPAM so not to drag Net struct around
	n.IPAM.Name = n.Name

	return n.IPAM, nil
}
