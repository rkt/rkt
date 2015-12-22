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

package aci

import (
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
)

type Type int

const (
	None Type = iota
	Basic
	Oauth
)

type httpError struct {
	code    int
	message string
}

func (e *httpError) Error() string {
	return fmt.Sprintf("%d: %s", e.code, e.message)
}

type serverHandler struct {
	auth    Type
	msg     chan<- string
	fileSet map[string]string
}

func (h *serverHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if authOk := h.handleAuth(w, r); !authOk {
		return
	}
	h.sendMsg(fmt.Sprintf("Trying to serve %q", r.URL.String()))
	h.handleRequest(w, r)
}

func (h *serverHandler) handleAuth(w http.ResponseWriter, r *http.Request) bool {
	switch h.auth {
	case None:
		// no auth to do.
		return true
	case Basic:
		return h.handleBasicAuth(w, r)
	case Oauth:
		return h.handleOauthAuth(w, r)
	default:
		panic("Woe is me!")
	}
}

func (h *serverHandler) handleBasicAuth(w http.ResponseWriter, r *http.Request) bool {
	payload, httpErr := getAuthPayload(r, "Basic")
	if httpErr != nil {
		w.WriteHeader(httpErr.code)
		h.sendMsg(fmt.Sprintf(`No "Authorization" header: %v`, httpErr.message))
		return false
	}
	creds, err := base64.StdEncoding.DecodeString(string(payload))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		h.sendMsg(`Badly formed "Authorization" header`)
		return false
	}
	parts := strings.Split(string(creds), ":")
	if len(parts) != 2 {
		w.WriteHeader(http.StatusBadRequest)
		h.sendMsg(`Badly formed "Authorization" header (2)`)
		return false
	}
	user := parts[0]
	password := parts[1]
	if user != "bar" || password != "baz" {
		w.WriteHeader(http.StatusUnauthorized)
		h.sendMsg(fmt.Sprintf("Bad credentials: %q", string(creds)))
		return false
	}
	return true
}

func (h *serverHandler) handleOauthAuth(w http.ResponseWriter, r *http.Request) bool {
	payload, httpErr := getAuthPayload(r, "Bearer")
	if httpErr != nil {
		w.WriteHeader(httpErr.code)
		h.sendMsg(fmt.Sprintf(`No "Authorization" header: %v`, httpErr.message))
		return false
	}
	if payload != "sometoken" {
		w.WriteHeader(http.StatusUnauthorized)
		h.sendMsg(fmt.Sprintf(`Bad token: %q`, payload))
		return false
	}
	return true
}

func getAuthPayload(r *http.Request, authType string) (string, *httpError) {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		err := &httpError{
			code:    http.StatusUnauthorized,
			message: "No auth",
		}
		return "", err
	}
	parts := strings.Split(auth, " ")
	if len(parts) != 2 {
		err := &httpError{
			code:    http.StatusBadRequest,
			message: "Malformed auth",
		}
		return "", err
	}
	if parts[0] != authType {
		err := &httpError{
			code:    http.StatusUnauthorized,
			message: "Wrong auth",
		}
		return "", err
	}
	return parts[1], nil
}

func (h *serverHandler) handleRequest(w http.ResponseWriter, r *http.Request) {
	path := filepath.Base(r.URL.Path)
	switch path {
	case "/":
		h.sendAcDiscovery(w)
		h.sendMsg("  done.")
	default:
		if found := h.handleFile(w, path); found {
			h.sendMsg("  done.")
		}
	}
}

func (h *serverHandler) sendAcDiscovery(w http.ResponseWriter) {
	// TODO(krnowak): When appc spec gets the discovery over
	// custom port feature, possibly take it into account here
	indexHTML := `<meta name="ac-discovery" content="localhost https://localhost/{name}.{ext}">`
	w.Write([]byte(indexHTML))
}

func (h *serverHandler) handleFile(w http.ResponseWriter, reqPath string) bool {
	path, ok := h.fileSet[reqPath]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		h.sendMsg("  not found.")
		return false
	}
	contents, err := ioutil.ReadFile(path)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		h.sendMsg("  not found, but specified in fileset; bug?")
		return false
	}
	w.Write(contents)
	return true
}

func (h *serverHandler) sendMsg(msg string) {
	select {
	case h.msg <- msg:
	default:
	}
}

type Server struct {
	Msg     <-chan string
	Conf    string
	URL     string
	handler *serverHandler
	http    *httptest.Server
}

func (s *Server) Close() {
	s.http.Close()
	close(s.handler.msg)
}

func (s *Server) UpdateFileSet(fileSet map[string]string) {
	s.handler.fileSet = fileSet
}

func NewServer(auth Type, msgCapacity int) *Server {
	msg := make(chan string, msgCapacity)
	server := &Server{
		Msg: msg,
		handler: &serverHandler{
			auth:    auth,
			msg:     msg,
			fileSet: make(map[string]string),
		},
	}
	server.http = httptest.NewUnstartedServer(server.handler)
	server.http.TLS = &tls.Config{InsecureSkipVerify: true}
	server.http.StartTLS()
	server.URL = server.http.URL
	host := server.http.Listener.Addr().String()
	switch auth {
	case None:
		// nothing to do
	case Basic:
		creds := `"user": "bar",
		"password": "baz"`
		server.Conf = sprintCreds(host, "basic", creds)
	case Oauth:
		creds := `"token": "sometoken"`
		server.Conf = sprintCreds(host, "oauth", creds)
	default:
		panic("Woe is me!")
	}
	return server
}

func sprintCreds(host, auth, creds string) string {
	return fmt.Sprintf(`
{
	"rktKind": "auth",
	"rktVersion": "v1",
	"domains": ["%s"],
	"type": "%s",
	"credentials":
	{
		%s
	}
}

`, host, auth, creds)
}
