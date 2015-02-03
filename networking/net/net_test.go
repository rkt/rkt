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

package net

import (
	"encoding/json"
	"io/ioutil"
	"reflect"
	"testing"
)

func saveToTemp(v interface{}) (string, error) {
	f, err := ioutil.TempFile("", "net")
	if err != nil {
		return "", err
	}
	defer f.Close()

	return f.Name(), json.NewEncoder(f).Encode(v)
}

func TestNet(t *testing.T) {
	expected := Net{
		Name: "mynet",
		Type: "veth",
	}
	expected.IPAlloc.Type = "static"
	expected.IPAlloc.Subnet = "10.1.2.0/24"

	fn, err := saveToTemp(expected)
	if err != nil {
		t.Fatal(err)
	}
	expected.Filename = fn

	actual := Net{}
	if err = LoadNet(fn, &actual); err != nil {
		t.Fatal(err)
	}

	if expected.Filename != actual.Filename {
		t.Errorf("Filename mismatch: expected=%q; actual=%q", expected.Filename, actual.Filename)
	}

	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("Mismatch: expected=%#v; actual=%#v", expected, actual)
	}
}

type MyNet struct {
	Net
}

func TestNetEmbedded(t *testing.T) {
	expected := MyNet{
		Net: Net{
			Name: "mynet",
			Type: "veth",
		},
	}
	expected.IPAlloc.Type = "static"
	expected.IPAlloc.Subnet = "10.1.2.0/24"

	fn, err := saveToTemp(expected)
	if err != nil {
		t.Fatal(err)
	}
	expected.Filename = fn

	actual := MyNet{}
	if err = LoadNet(fn, &actual); err != nil {
		t.Fatal(err)
	}

	if expected.Filename != actual.Filename {
		t.Errorf("Filename mismatch: expected=%q; actual=%q", expected.Filename, actual.Filename)
	}

	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("Mismatch: expected=%#v; actual=%#v", expected, actual)
	}
}
