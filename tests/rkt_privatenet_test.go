// Copyright 2015 The rkt Authors
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

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/coreos/rkt/Godeps/_workspace/src/github.com/steveeJ/gexpect"
	"github.com/coreos/rkt/tests/testutils"
)

/*
 * No private net
 * ---
 * Container must have the same network namespace as the host
 */
func TestPrivateNetOmittedNetNS(t *testing.T) {
	testImageArgs := []string{"--exec=/inspect --print-netns"}
	testImage := patchTestACI("rkt-inspect-networking.aci", testImageArgs...)
	defer os.Remove(testImage)

	ctx := newRktRunCtx()
	defer ctx.cleanup()
	defer ctx.reset()

	cmd := fmt.Sprintf("%s --debug --insecure-skip-verify run --mds-register=false %s", ctx.cmd(), testImage)
	t.Logf("Command: %v\n", cmd)
	child, err := gexpect.Spawn(cmd)
	if err != nil {
		t.Fatalf("Cannot exec rkt: %v", err)
	}

	expectedRegex := `NetNS: (net:\[\d+\])`
	result, out, err := expectRegexWithOutput(child, expectedRegex)
	if err != nil {
		t.Fatalf("Error: %v\nOutput: %v", err, out)
	}

	ns, err := os.Readlink("/proc/self/ns/net")
	if err != nil {
		t.Fatalf("Cannot evaluate NetNS symlink: %v", err)
	}

	if nsChanged := ns != result[1]; nsChanged {
		t.Fatalf("container left host netns")
	}

	err = child.Wait()
	if err != nil {
		t.Fatalf("rkt didn't terminate correctly: %v", err)
	}
}

/*
 * No private net
 * ---
 * Container launches http server which must be reachable by the host via the
 * localhost address
 */
func TestPrivateNetOmittedConnectivity(t *testing.T) {
	httpServeAddr := "0.0.0.0:54321"
	httpGetAddr := "http://127.0.0.1:54321"

	testImageArgs := []string{"--exec=/inspect --serve-http=" + httpServeAddr}
	testImage := patchTestACI("rkt-inspect-networking.aci", testImageArgs...)
	defer os.Remove(testImage)

	ctx := newRktRunCtx()
	defer ctx.cleanup()
	defer ctx.reset()

	cmd := fmt.Sprintf("%s --debug --insecure-skip-verify run --mds-register=false %s", ctx.cmd(), testImage)
	t.Logf("Command: %v\n", cmd)
	child, err := gexpect.Spawn(cmd)
	if err != nil {
		t.Fatalf("Cannot exec rkt: %v", err)
	}

	// Child opens the server
	c := make(chan struct{})
	go func() {
		err = child.Wait()
		if err != nil {
			t.Fatalf("rkt didn't terminate correctly: %v", err)
		}
		c <- struct{}{}
	}()

	// Host connects to the child
	go func() {
		expectedRegex := `serving on`
		_, out, err := expectRegexWithOutput(child, expectedRegex)
		if err != nil {
			t.Fatalf("Error: %v\nOutput: %v", err, out)
		}
		body, err := testutils.HttpGet(httpGetAddr)
		if err != nil {
			log.Fatalf("%v\n", err)
		}
		log.Printf("HTTP-Get received: %s", body)
	}()

	<-c
}

/*
 * Default private-net
 * ---
 * Container must be in a separate network namespace with private-net
 */
func TestPrivateNetDefaultNetNS(t *testing.T) {
	testImageArgs := []string{"--exec=/inspect --print-netns"}
	testImage := patchTestACI("rkt-inspect-networking.aci", testImageArgs...)
	defer os.Remove(testImage)

	ctx := newRktRunCtx()
	defer ctx.cleanup()
	defer ctx.reset()

	cmd := fmt.Sprintf("%s --debug --insecure-skip-verify run --private-net=default --mds-register=false %s", ctx.cmd(), testImage)
	t.Logf("Command: %v\n", cmd)
	child, err := gexpect.Spawn(cmd)
	if err != nil {
		t.Fatalf("Cannot exec rkt: %v", err)
	}

	expectedRegex := `NetNS: (net:\[\d+\])`
	result, out, err := expectRegexWithOutput(child, expectedRegex)
	if err != nil {
		t.Fatalf("Error: %v\nOutput: %v", err, out)
	}

	ns, err := os.Readlink("/proc/self/ns/net")
	if err != nil {
		t.Fatalf("Cannot evaluate NetNS symlink: %v", err)
	}

	if nsChanged := ns != result[1]; !nsChanged {
		t.Fatalf("container did not leave host netns")
	}

	err = child.Wait()
	if err != nil {
		t.Fatalf("rkt didn't terminate correctly: %v", err)
	}
}

/*
 * Default private-net
 * ---
 * Host launches http server on all interfaces in the host netns
 * Container must be able to connect via any IP address of the host in the
 * default network, which is NATed
 */
func TestPrivateNetDefaultConnectivity(t *testing.T) {
	httpServeAddr := "0.0.0.0:54321"
	httpServeTimeout := 30

	ifaces, err := net.Interfaces()
	if err != nil {
		t.Fatalf("Cannot get network host's interfaces: %v", err)
	}
	var httpGetAddr string
	for _, iface := range ifaces[1:] {
		name := iface.Name
		ifaceIPsv4, err := testutils.GetIPsv4(name)
		if err != nil {
			t.Fatalf("Cannot get IPV4 address for interface %v: %v", name, err)
		}
		if len(ifaceIPsv4) > 0 {
			httpGetAddr = fmt.Sprintf("http://%v:54321", ifaceIPsv4[0])
			t.Log("Telling the child to connect via", httpGetAddr)
			break
		}
	}
	if httpGetAddr == "" {
		t.Skipf("Can not find any NAT'able IPv4 on the host, skipping..")
	}

	testImageArgs := []string{fmt.Sprintf("--exec=/inspect --get-http=%v", httpGetAddr)}
	testImage := patchTestACI("rkt-inspect-networking.aci", testImageArgs...)
	defer os.Remove(testImage)

	ctx := newRktRunCtx()
	defer ctx.cleanup()
	defer ctx.reset()

	var wg sync.WaitGroup

	// Host opens the server
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := test_netutils.HttpServe(httpServeAddr, httpServeTimeout)
		if err != nil {
			t.Fatalf("Error during HttpServe: %v", err)
		}
	}()

	// Child connects to host
	wg.Add(1)
	hostname, err := os.Hostname()
	go func() {
		defer wg.Done()
		cmd := fmt.Sprintf("%s --debug --insecure-skip-verify run --private-net=default --mds-register=false %s", ctx.cmd(), testImage)
		t.Logf("Command: %v\n", cmd)
		child, err := gexpect.Spawn(cmd)
		if err != nil {
			t.Fatalf("Cannot exec rkt: %v", err)
		}
		expectedRegex := `HTTP-Get received: (.*)\r`
		result, out, err := expectRegexWithOutput(child, expectedRegex)
		if err != nil {
			t.Fatalf("Error: %v\nOutput: %v", err, out)
		}
		if result[1] != hostname {
			t.Fatalf("Hostname received by client `%v` doesn't match `%v`", result[1], hostname)
		}

		err = child.Wait()
		if err != nil {
			t.Fatalf("rkt didn't terminate correctly: %v", err)
		}
	}()

	wg.Wait()

}

/*
 * Default-restricted private-net
 * ---
 * Container launches http server on all its interfaces
 * Host must be able to connects to container's http server via container's
 * eth0's IPv4
 * TODO: verify that the container isn't NATed
 */
func TestPrivateNetDefaultRestrictedConnectivity(t *testing.T) {
	httpServeAddr := "0.0.0.0:54321"
	iface := "eth0"

	testImageArgs := []string{fmt.Sprintf("--exec=/inspect --print-ipv4=%v --serve-http=%v", iface, httpServeAddr)}
	testImage := patchTestACI("rkt-inspect-networking.aci", testImageArgs...)
	defer os.Remove(testImage)

	ctx := newRktRunCtx()
	defer ctx.cleanup()
	defer ctx.reset()

	cmd := fmt.Sprintf("%s --debug --insecure-skip-verify run --private-net=default-restricted --mds-register=false %s", ctx.cmd(), testImage)
	t.Logf("Command: %v\n", cmd)
	child, err := gexpect.Spawn(cmd)
	if err != nil {
		t.Fatalf("Cannot exec rkt: %v", err)
	}

	expectedRegex := `IPv4: (.*)\r`
	result, out, err := expectRegexWithOutput(child, expectedRegex)
	if err != nil {
		t.Fatalf("Error: %v\nOutput: %v", err, out)
	}
	httpGetAddr := fmt.Sprintf("http://%v:54321", result[1])

	var wg sync.WaitGroup

	// Child opens the server
	wg.Add(1)
	go func() {
		defer wg.Done()
		err = child.Wait()
		if err != nil {
			t.Fatalf("rkt didn't terminate correctly: %v", err)
		}
	}()

	// Host connects to the child
	wg.Add(1)
	go func() {
		defer wg.Done()
		expectedRegex := `serving on`
		_, out, err := expectRegexWithOutput(child, expectedRegex)
		if err != nil {
			t.Fatalf("Error: %v\nOutput: %v", err, out)
		}
		body, err := testutils.HttpGet(httpGetAddr)
		if err != nil {
			log.Fatalf("%v\n", err)
		}
		log.Printf("HTTP-Get received: %s", body)
		if err != nil {
			log.Fatalf("%v\n", err)
		}
	}()

	wg.Wait()
}

func writeNetwork(t *testing.T, net networkTemplateT, netd string) error {
	var err error
	path := filepath.Join(netd, net.Name+".conf")
	file, err := os.Create(path)
	if err != nil {
		t.Errorf("%v", err)
	}

	b, err := json.Marshal(net)
	if err != nil {
		return err
	}

	fmt.Println("Writing", net.Name, "to", path)
	_, err = file.Write(b)
	if err != nil {
		return err
	}

	return nil
}

var networkTemplate string = `
{
"name": "{{.Name}}",
"type": "{{.Typ}}",
{{ if .Master }}"master": "{{.Master}}",{{end}}
"ipam": {
	"type": "{{.IpamTyp}}",
	{{ if .IpamSubnet }}"subnet": "{{.IpamSubnet}}",{{end}}
	{{ if .IpamRoutes }}"routes": {{.IpamRoutes}}{{end}}
	}
}
`

type networkTemplateT struct {
	Name   string
	Type   string
	Master string `json:"master,omitempty"`
	IpMasq bool
	Ipam   ipamTemplateT
}

type ipamTemplateT struct {
	Type   string
	Subnet string              `json:"subnet,omitempty"`
	Routes []map[string]string `json:"routes,omitempty"`
}

func TestTemplates(t *testing.T) {
	net := networkTemplateT{
		Name: "ptp0",
		Type: "ptp",
		Ipam: ipamTemplateT{
			Type:   "host-local-ptp",
			Subnet: "10.1.3.0/24",
			Routes: []map[string]string{{"dst": "0.0.0.0/0"}},
		},
	}

	b, err := json.Marshal(net)
	if err != nil {
		t.Fatalf("%v", err)
	}
	t.Logf("%v", string(b))
}

/*
 * Two containers spawn in the same custom network.
 * ---
 * Container 1 opens the http server
 * Container 2 fires a HttpGet on it
 * The body of the HttpGet is Container 1's hostname, which must match
 */
func testPrivateNetCustomDual(t *testing.T, net networkTemplateT) {
	ctx := newRktRunCtx()
	defer ctx.cleanup()
	defer ctx.reset()

	configdir := ctx.directories[1].dir
	netdir := filepath.Join(configdir, "net.d")
	err := os.MkdirAll(netdir, 0644)
	defer os.RemoveAll(netdir)
	if err != nil {
		t.Fatalf("Cannot create netdir: %v", err)
	}
	err = writeNetwork(t, net, netdir)
	if err != nil {
		t.Fatalf("Cannot write network file: %v", err)
	}

	container1IPv4, container1Hostname, abort := make(chan string), make(chan string), make(chan bool)
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		httpServeAddr := "0.0.0.0:54321"
		testImageArgs := []string{"--exec=/inspect --print-ipv4=eth0 --serve-http=" + httpServeAddr}
		testImage := patchTestACI("rkt-inspect-networking1.aci", testImageArgs...)
		defer os.Remove(testImage)

		cmd := fmt.Sprintf("%s --debug --insecure-skip-verify run --private-net=%v --mds-register=false %s", ctx.cmd(), net.Name, testImage)
		fmt.Printf("Command: %v\n", cmd)
		child, err := gexpect.Spawn(cmd)
		if err != nil {
			log.Fatalf("Cannot exec rkt: %v", err)
		}

		expectedRegex := `IPv4: (\d+\.\d+\.\d+\.\d+)`
		result, out, err := expectRegexTimeoutWithOutput(child, expectedRegex, 30*time.Second)
		if err != nil {
			abort <- true
			t.Fatalf("Error: %v\nOutput: %v", err, out)
		} else {
			container1IPv4 <- result[1]
			expectedRegex = `(rkt-.*): serving on`
			result, out, err = expectRegexTimeoutWithOutput(child, expectedRegex, 30*time.Second)
			if err != nil {
				t.Fatalf("Error: %v\nOutput: %v", err, out)
				abort <- true
			} else {
				container1Hostname <- result[1]
			}
		}
		err = child.Wait()
		if err != nil {
			t.Fatalf("rkt didn't terminate correctly: %v", err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		var httpGetAddr string
		select {
		case <-abort:
			return
		case ipv4 := <-container1IPv4:
			httpGetAddr = fmt.Sprintf("http://%v:54321", ipv4)
		}

		testImageArgs := []string{"--exec=/inspect --get-http=" + httpGetAddr}
		testImage := patchTestACI("rkt-inspect-networking2.aci", testImageArgs...)
		defer os.Remove(testImage)

		cmd := fmt.Sprintf("%s --debug --insecure-skip-verify run --private-net=%v --mds-register=false %s", ctx.cmd(), net.Name, testImage)
		fmt.Printf("Command: %v\n", cmd)
		child, err := gexpect.Spawn(cmd)
		if err != nil {
			t.Fatalf("Cannot exec rkt: %v", err)
		}

		select {
		case <-abort:
		case expectedHostname := <-container1Hostname:
			expectedRegex := `HTTP-Get received: (.*)\r`
			result, out, err := expectRegexTimeoutWithOutput(child, expectedRegex, 20*time.Second)
			if err != nil {
				t.Fatalf("Error: %v\nOutput: %v", err, out)
			}
			t.Logf("HTTP-Get received: %s", result[1])
			receivedHostname := result[1]

			if receivedHostname != expectedHostname {
				t.Fatalf("Received hostname `%v` doesn't match `%v`", receivedHostname, expectedHostname)
			}
		}

		err = child.Wait()
		if err != nil {
			t.Fatalf("rkt didn't terminate correctly: %v", err)
		}
	}()

	wg.Wait()
}

func TestPrivateNetCustomPtp(t *testing.T) {
	net := networkTemplateT{
		Name:   "ptp0",
		Type:   "ptp",
		IpMasq: false,
		Ipam: ipamTemplateT{
			Type:   "host-local-ptp",
			Subnet: "10.1.3.0/24",
			Routes: []map[string]string{
				{"dst": "0.0.0.0/0"},
			},
		},
	}
	testPrivateNetCustomDual(t, net)
}

/*
 * TODO: test connection to host on an outside interface
 */
func TestPrivateNetCustomMacvlan(t *testing.T) {
	ifaces, err := net.Interfaces()
	var ifaceName string
	if err != nil {
		t.Fatalf("Cannot get network host's interfaces: %v", err)
	} else if len(ifaces) >= 2 {
		ifaceName = ifaces[1].Name
	}
	net := networkTemplateT{
		Name:   "macvlan0",
		Type:   "macvlan",
		Master: ifaceName,
		Ipam: ipamTemplateT{Type: "host-local",
			Subnet: "10.1.2.0/24",
		},
	}
	testPrivateNetCustomDual(t, net)
}
