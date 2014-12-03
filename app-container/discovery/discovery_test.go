package discovery

import (
	"net/http"
	"os"
	"testing"
)

func fakeHTTPGet(filename string) func(uri string) (*http.Response, error) {
	return func(uri string) (*http.Response, error) {
		f, err := os.Open(filename)
		if err != nil {
			return nil, err
		}

		return &http.Response{
			Status:     "200 OK",
			StatusCode: 200,
			Proto:      "HTTP/1.1",
			ProtoMajor: 1,
			ProtoMinor: 1,
			Header: http.Header{
				"Content-Type": []string{"text/html"},
			},
			Body: f,
		}, nil
	}
}

func TestDiscoverEndpoints(t *testing.T) {
	httpGet = fakeHTTPGet("myapp.html")

	a := App{
		Name: "example.com/myapp",
		Labels: map[string]string{
			"version": "1.0.0",
			"os":      "linux",
			"arch":    "amd64",
		},
	}
	de, err := DiscoverEndpoints(a, true)
	if err != nil {
		t.Fatal(err)
	}

	if len(de.Sig) != 2 {
		t.Errorf("Sig array is wrong length")
	} else {
		if de.Sig[0] != "https://storage.example.com/example.com/myapp-1.0.0.sig?torrent" {
			t.Error("Sig[0] mismatch: ", de.Sig[0])
		}
		if de.Sig[1] != "hdfs://storage.example.com/example.com/myapp-1.0.0.sig" {
			t.Error("Sig[1] mismatch: ", de.Sig[0])
		}
	}

	if len(de.ACI) != 2 {
		t.Errorf("ACI array is wrong length")
	} else {
		if de.ACI[0] != "https://storage.example.com/example.com/myapp-1.0.0.aci?torrent" {
			t.Error("ACI[0] mismatch: ", de.ACI[0])
		}
		if de.ACI[1] != "hdfs://storage.example.com/example.com/myapp-1.0.0.aci" {
			t.Error("ACI[1] mismatch: ", de.ACI[1])
		}
	}

	if len(de.Keys) != 1 {
		t.Errorf("Keys array is wrong length")
	} else {
		if de.Keys[0] != "https://example.com/pubkeys.gpg" {
			t.Error("Keys[0] mismatch: ", de.Keys[0])
		}
	}
}
