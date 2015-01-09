package util

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
