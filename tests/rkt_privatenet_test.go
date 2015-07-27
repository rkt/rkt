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
	"fmt"
	"log"
	"net"
	"os"
	"sync"
	"testing"

	"github.com/coreos/rkt/Godeps/_workspace/src/github.com/ThomasRooney/gexpect"
	"github.com/coreos/rkt/tests/test_netutils"
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
	t.Logf("Command: %v", cmd)
	child, err := gexpect.Spawn(cmd)
	if err != nil {
		t.Fatalf("Cannot exec rkt: %v", err)
	}

	expectedRegex := `NetNS: (net:\[\d+\])`
	result, _ := child.ExpectRegexFind(expectedRegex)
	if len(result) == 0 {
		t.Fatalf("Expected %q but not found", expectedRegex)
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
	t.Logf("Command: %v", cmd)
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
		result, _ := child.ExpectRegexFind(expectedRegex)
		if len(result) == 0 {
			t.Fatalf("Expected %q but not found", expectedRegex)
		}
		body, err := test_netutils.HttpGet(httpGetAddr)
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

	cmd := fmt.Sprintf("%s --debug --insecure-skip-verify run --private-net=default --mds-register=false %s", ctx.cmd(), testImage)
	t.Logf("Command: %v", cmd)
	child, err := gexpect.Spawn(cmd)
	if err != nil {
		t.Fatalf("Cannot exec rkt: %v", err)
	}

	expectedRegex := `NetNS: (net:\[\d+\])`
	result, _ := child.ExpectRegexFind(expectedRegex)
	if len(result) == 0 {
		t.Fatalf("Expected %q but not found", expectedRegex)
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

	ctx.reset()
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
	iface := ifaces[1].Name
	ifaceIPsv4, err := test_netutils.GetIPsv4(iface)
	if err != nil {
		t.Fatalf("Cannot get network host's default IPv4: %v", err)
	}
	httpGetAddr := fmt.Sprintf("http://%v:54321", ifaceIPsv4[0])

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
		t.Logf("Command: %v", cmd)
		child, err := gexpect.Spawn(cmd)
		if err != nil {
			t.Fatalf("Cannot exec rkt: %v", err)
		}
		expectedRegex := `HTTP-Get received: (.*)\r`
		result, _ := child.ExpectRegexFind(expectedRegex)
		if len(result) == 0 {
			t.Fatalf("Expected %q but not found", expectedRegex)
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
	t.Logf("Command: %v", cmd)
	child, err := gexpect.Spawn(cmd)
	if err != nil {
		t.Fatalf("Cannot exec rkt: %v", err)
	}

	expectedRegex := `IPv4: (.*)\r`
	result, _ := child.ExpectRegexFind(expectedRegex)
	if len(result) == 0 {
		t.Fatalf("Expected %q but not found", expectedRegex)
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
		result, _ := child.ExpectRegexFind(expectedRegex)
		if len(result) == 0 {
			t.Fatalf("Expected %q but not found", expectedRegex)
		}
		body, err := test_netutils.HttpGet(httpGetAddr)
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
