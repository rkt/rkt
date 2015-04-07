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

package config

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
)

const (
	authHeader string = "Authorization"
)

type authV1JsonParser struct{}

type authV1 struct {
	Domains     []string        `json:"domains"`
	Type        string          `json:"type"`
	Credentials json.RawMessage `json:"credentials"`
}

type basicV1 struct {
	User     string `json:"user"`
	Password string `json:"password"`
}

type oauthV1 struct {
	Token string `json:"token"`
}

func init() {
	addParser("auth", "v1", &authV1JsonParser{})
	registerSubDir("auth.d", []string{"auth"})
}

type BasicAuthHeaderer struct {
	user     string
	password string
}

func (h *BasicAuthHeaderer) Header() http.Header {
	headers := make(http.Header)
	creds := []byte(fmt.Sprintf("%s:%s", h.user, h.password))
	encodedCreds := base64.StdEncoding.EncodeToString(creds)
	headers.Add(authHeader, "Basic "+encodedCreds)

	return headers
}

type OAuthBearerTokenHeaderer struct {
	token string
}

func (h *OAuthBearerTokenHeaderer) Header() http.Header {
	headers := make(http.Header)
	headers.Add(authHeader, "Bearer "+h.token)

	return headers
}

func (p *authV1JsonParser) parse(config *Config, raw []byte) error {
	var auth authV1
	if err := json.Unmarshal(raw, &auth); err != nil {
		return err
	}
	if len(auth.Domains) == 0 {
		return fmt.Errorf("No domains specified")
	}
	if len(auth.Type) == 0 {
		return fmt.Errorf("No auth type specified")
	}
	var (
		err      error
		headerer Headerer
	)
	switch auth.Type {
	case "basic":
		headerer, err = p.getBasicV1Headerer(config, auth.Credentials)
	case "oauth":
		headerer, err = p.getOAuthV1Headerer(config, auth.Credentials)
	default:
		err = fmt.Errorf("Unknown auth type: %q", auth.Type)
	}
	if err != nil {
		return err
	}
	for _, domain := range auth.Domains {
		if _, ok := config.AuthPerHost[domain]; ok {
			return fmt.Errorf("auth for domain %q is already specified", domain)
		}
		config.AuthPerHost[domain] = headerer
	}
	return nil
}

func (p *authV1JsonParser) getBasicV1Headerer(config *Config, raw json.RawMessage) (Headerer, error) {
	var basic basicV1
	if err := json.Unmarshal(raw, &basic); err != nil {
		return nil, err
	}
	if len(basic.User) == 0 {
		return nil, fmt.Errorf("User not specified")
	}
	if len(basic.Password) == 0 {
		return nil, fmt.Errorf("Password not specified")
	}
	return &BasicAuthHeaderer{
		user:     basic.User,
		password: basic.Password,
	}, nil
}

func (p *authV1JsonParser) getOAuthV1Headerer(config *Config, raw json.RawMessage) (Headerer, error) {
	var oauth oauthV1
	if err := json.Unmarshal(raw, &oauth); err != nil {
		return nil, err
	}
	if len(oauth.Token) == 0 {
		return nil, fmt.Errorf("No oauth bearer token specified")
	}
	return &OAuthBearerTokenHeaderer{
		token: oauth.Token,
	}, nil
}
