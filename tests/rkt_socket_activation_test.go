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
	"bufio"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	sd_dbus "github.com/coreos/rkt/Godeps/_workspace/src/github.com/coreos/go-systemd/dbus"
	sd_util "github.com/coreos/rkt/Godeps/_workspace/src/github.com/coreos/go-systemd/util"
)

func randomFreePort(t *testing.T) (int, error) {
	l, err := net.Listen("tcp", "")
	if err != nil {
		return -1, err
	}
	defer l.Close()

	addr := l.Addr().String()

	n := strings.LastIndex(addr, ":")
	port, err := strconv.Atoi(addr[n+1:])
	if err != nil {
		return -1, err
	}

	return port, nil
}

func TestSocketActivation(t *testing.T) {
	if !sd_util.IsRunningSystemd() {
		t.Skip("Systemd is not running on the host.")
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	port, err := randomFreePort(t)
	if err != nil {
		t.Fatal(err)
	}

	echoImage := patchTestACI("rkt-inspect-echo.aci",
		"--exec=/echo-socket-activated",
		fmt.Sprintf("--ports=%d-tcp,protocol=tcp,port=%d,socketActivated=true", port, port))
	defer os.Remove(echoImage)

	ctx := newRktRunCtx()
	defer ctx.cleanup()

	conn, err := sd_dbus.New()
	if err != nil {
		t.Fatal(err)
	}

	rktTestingEchoService := `
	[Unit]
	Description=Socket-activated echo server

	[Service]
	ExecStart=%s
	KillMode=process
	`

	rnd := r.Int()

	cmd := fmt.Sprintf("%s --insecure-skip-verify run --mds-register=false %s", ctx.cmd(), echoImage)
	serviceContent := fmt.Sprintf(rktTestingEchoService, cmd)
	serviceTarget := fmt.Sprintf("rkt-testing-socket-activation-%d.service", rnd)

	if err := ioutil.WriteFile(serviceTarget, []byte(serviceContent), 0666); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(serviceTarget)

	rktTestingEchoSocket := `
	[Unit]
	Description=Socket-activated netcat server socket

	[Socket]
	ListenStream=%d
	`
	socketContent := fmt.Sprintf(rktTestingEchoSocket, port)
	socketTarget := fmt.Sprintf("rkt-testing-socket-activation-%d.socket", rnd)

	if err := ioutil.WriteFile(socketTarget, []byte(socketContent), 0666); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(socketTarget)

	serviceTargetAbs, err := filepath.Abs(serviceTarget)
	if err != nil {
		t.Fatal(err)
	}

	_, err = conn.LinkUnitFiles([]string{serviceTargetAbs}, true, false)
	if err != nil {
		t.Fatal(err)
	}

	socketTargetAbs, err := filepath.Abs(socketTarget)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := conn.LinkUnitFiles([]string{socketTargetAbs}, true, false); err != nil {
		t.Fatal(err)
	}

	reschan := make(chan string)
	doJob := func() {
		job := <-reschan
		if job != "done" {
			t.Fatal("Job is not done:", job)
		}
	}

	if _, err := conn.StartUnit(socketTarget, "replace", reschan); err != nil {
		t.Fatal(err)
	}
	doJob()

	defer func() {
		if _, err := conn.StopUnit(socketTarget, "replace", reschan); err != nil {
			t.Fatal(err)
		}
		doJob()

		if _, err := conn.StopUnit(serviceTarget, "replace", reschan); err != nil {
			t.Fatal(err)
		}
		doJob()
	}()

	expected := "HELO\n"
	sockConn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", port))
	if err != nil {
		t.Fatal(err)
	}

	if _, err := fmt.Fprintf(sockConn, expected); err != nil {
		t.Fatal(err)
	}

	answer, err := bufio.NewReader(sockConn).ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}

	if answer != expected {
		t.Fatalf("Expected %q, Got %q", expected, answer)
	}

	return
}
