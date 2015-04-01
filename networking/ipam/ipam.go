package ipam

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/coreos/rkt/Godeps/_workspace/src/github.com/vishvananda/netlink"

	"github.com/coreos/rkt/networking/util"
)

// L3 config value for interface
type IPConfig struct {
	IP      *net.IPNet
	Gateway net.IP
	Routes  []net.IPNet
}

type ipConfig struct {
	IP      string   `json:"ip"`
	Gateway string   `json:"gateway,omitempty"`
	Routes  []string `json:"routes,omitempty"`
}

func (c *IPConfig) UnmarshalJSON(data []byte) error {
	ipc := ipConfig{}
	if err := json.Unmarshal(data, &ipc); err != nil {
		return err
	}

	ip, err := util.ParseCIDR(ipc.IP)
	if err != nil {
		return err
	}

	var gw net.IP
	if ipc.Gateway != "" {
		if gw = net.ParseIP(ipc.Gateway); gw == nil {
			return fmt.Errorf("error parsing Gateway")
		}
	}

	routes := []net.IPNet{}

	for _, r := range ipc.Routes {
		dst, err := util.ParseCIDR(r)
		if err != nil {
			return err
		}

		routes = append(routes, *dst)
	}

	c.IP = ip
	c.Gateway = gw
	c.Routes = routes

	return nil
}

func (c *IPConfig) MarshalJSON() ([]byte, error) {
	ipc := ipConfig{
		IP: c.IP.String(),
	}

	if c.Gateway != nil {
		ipc.Gateway = c.Gateway.String()
	}

	for _, dst := range c.Routes {
		ipc.Routes = append(ipc.Routes, dst.String())
	}

	return json.Marshal(ipc)
}

func findIPAMPlugin(plugin string) string {
	// try 3rd-party path first
	paths := strings.Split(os.Getenv("RKT_NETPLUGIN_IPAMPATH"), ":")

	for _, p := range paths {
		fullname := filepath.Join(p, plugin)
		if fi, err := os.Stat(fullname); err == nil && fi.Mode().IsRegular() {
			return fullname
		}
	}

	return ""
}

// Executes IPAM plugin, assuming RKT_NETPLUGIN_COMMAND == ADD.
// Parses and returns resulting IPConfig
func ExecPluginAdd(plugin string) (*IPConfig, error) {
	if os.Getenv("RKT_NETPLUGIN_COMMAND") != "ADD" {
		return nil, fmt.Errorf("RKT_NETPLUGIN_COMMAND is not ADD")
	}

	pluginPath := findIPAMPlugin(plugin)
	if pluginPath == "" {
		return nil, fmt.Errorf("could not find %q plugin", plugin)
	}

	stdout := &bytes.Buffer{}

	c := exec.Cmd{
		Path:   pluginPath,
		Args:   []string{pluginPath},
		Stdout: stdout,
		Stderr: os.Stderr,
	}
	if err := c.Run(); err != nil {
		return nil, err
	}

	ipConf := &IPConfig{}
	err := json.Unmarshal(stdout.Bytes(), ipConf)
	return ipConf, err
}

// Executes IPAM plugin, assuming RKT_NETPLUGIN_COMMAND == DEL.
func ExecPluginDel(plugin string) error {
	if os.Getenv("RKT_NETPLUGIN_COMMAND") != "DEL" {
		return fmt.Errorf("RKT_NETPLUGIN_COMMAND is not DEL")
	}

	pluginPath := findIPAMPlugin(plugin)
	if pluginPath == "" {
		return fmt.Errorf("could not find %q plugin", plugin)
	}

	c := exec.Cmd{
		Path:   pluginPath,
		Args:   []string{pluginPath},
		Stderr: os.Stderr,
	}
	return c.Run()
}

func ApplyIPConfig(ifName string, ipConf *IPConfig) error {
	link, err := netlink.LinkByName(ifName)
	if err != nil {
		return fmt.Errorf("failed to lookup %q: %v", ifName, err)
	}

	if err := netlink.LinkSetUp(link); err != nil {
		return fmt.Errorf("failed too set %q UP: %v", ifName, err)
	}

	addr := &netlink.Addr{IPNet: ipConf.IP, Label: ""}
	if err = netlink.AddrAdd(link, addr); err != nil {
		return fmt.Errorf("failed to add IP addr to %q: %v", ifName, err)
	}

	for _, dst := range ipConf.Routes {
		if err = util.AddRoute(&dst, ipConf.Gateway, link); err != nil {
			// we skip over duplicate routes as we assume the first one wins
			if !os.IsExist(err) {
				return fmt.Errorf("failed to add route '%v via %v dev %v': %v", dst.String(), ipConf.Gateway, ifName, err)
			}
		}
	}

	return nil
}
