package lib

import (
	"crypto/tls"
	"fmt"
	"net/http"
)

func StartServer(auth Type) (*Server, error) {
	return NewServer(auth, 10)
}

func StopServer(host string) (*http.Response, error) {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: transport}
	res, err := client.Post(host, "whatever", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to send post to %q: %v", host, err)
	}
	return res, nil
}
