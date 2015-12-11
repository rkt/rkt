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

package image

import (
	"bytes"
	"fmt"
	"io"
	"net/url"
	"time"

	"github.com/coreos/rkt/pkg/keystore"
	"github.com/coreos/rkt/rkt/config"
	rktflag "github.com/coreos/rkt/rkt/flag"
	"github.com/coreos/rkt/rkt/pubkey"
	"github.com/coreos/rkt/store"

	"github.com/coreos/rkt/Godeps/_workspace/src/github.com/appc/spec/discovery"
	pgperrors "github.com/coreos/rkt/Godeps/_workspace/src/golang.org/x/crypto/openpgp/errors"
)

// nameFetcher is used to download images via discovery
type nameFetcher struct {
	InsecureFlags      *rktflag.SecFlags
	S                  *store.Store
	Ks                 *keystore.Keystore
	Debug              bool
	Headers            map[string]config.Headerer
	TrustKeysFromHttps bool
}

// GetHash runs the discovery, fetches the image, optionally verifies
// it against passed asc, stores it in the store and returns the hash.
func (f *nameFetcher) GetHash(app *discovery.App, a *asc) (string, error) {
	name := app.Name.String()
	stderr("searching for app image %s", name)
	ep, err := f.discoverApp(app)
	if err != nil {
		return "", fmt.Errorf("discovery failed for %q: %v", name, err)
	}
	latest := false
	// No specified version label, mark it as latest
	if _, ok := app.Labels["version"]; !ok {
		latest = true
	}
	return f.fetchImageFromEndpoints(app, ep, a, latest)
}

func (f *nameFetcher) discoverApp(app *discovery.App) (*discovery.Endpoints, error) {
	// TODO(krnowak): Instead of hardcoding InsecureHttp, we probably
	// should use f.InsecureFlags.AllowHttp and
	// f.InsecureFlags.AllowHttpCredentials (if they are
	// introduced) on it. Needs some work first on appc/spec side.
	// https://github.com/appc/spec/issues/545
	// https://github.com/coreos/rkt/issues/1836
	insecure := discovery.InsecureHttp
	if f.InsecureFlags.SkipTlsCheck() {
		insecure = insecure | discovery.InsecureTls
	}
	hostHeaders := config.ResolveAuthPerHost(f.Headers)
	ep, attempts, err := discovery.DiscoverEndpoints(*app, hostHeaders, insecure)
	if f.Debug {
		for _, a := range attempts {
			stderr("meta tag 'ac-discovery' not found on %s: %v", a.Prefix, a.Error)
		}
	}
	if err != nil {
		return nil, err
	}
	if len(ep.ACIEndpoints) == 0 {
		return nil, fmt.Errorf("no endpoints discovered")
	}
	return ep, nil
}

func (f *nameFetcher) fetchImageFromEndpoints(app *discovery.App, ep *discovery.Endpoints, a *asc, latest bool) (string, error) {
	// TODO(krnowak): we should probably try all the endpoints,
	// for this we need to clone "a" and call
	// maybeOverrideAscFetcherWithRemote on the clone
	aciURL := ep.ACIEndpoints[0].ACI
	ascURL := ep.ACIEndpoints[0].ASC
	stderr("remote fetching from URL %q", aciURL)
	f.maybeOverrideAscFetcherWithRemote(ascURL, a)
	return f.fetchImageFromSingleEndpoint(app, aciURL, a, latest)
}

func (f *nameFetcher) fetchImageFromSingleEndpoint(app *discovery.App, aciURL string, a *asc, latest bool) (string, error) {
	if f.Debug {
		stderr("fetching image from %s", aciURL)
	}

	aciFile, cd, err := f.fetch(app, aciURL, a)
	if err != nil {
		return "", err
	}
	defer aciFile.Close()

	key, err := f.S.WriteACI(aciFile, latest)
	if err != nil {
		return "", err
	}

	rem := store.NewRemote(aciURL, a.Location)
	rem.BlobKey = key
	rem.DownloadTime = time.Now()
	rem.ETag = cd.ETag
	rem.CacheMaxAge = cd.MaxAge
	err = f.S.WriteRemote(rem)
	if err != nil {
		return "", err
	}

	return key, nil
}

func (f *nameFetcher) fetch(app *discovery.App, aciURL string, a *asc) (readSeekCloser, *cacheData, error) {
	if f.InsecureFlags.SkipTlsCheck() && f.Ks != nil {
		stderr("warning: TLS verification has been disabled")
	}
	if f.InsecureFlags.SkipImageCheck() && f.Ks != nil {
		stderr("warning: image signature verification has been disabled")
	}

	u, err := url.Parse(aciURL)
	if err != nil {
		return nil, nil, fmt.Errorf("error parsing ACI url: %v", err)
	}

	if f.InsecureFlags.SkipImageCheck() || f.Ks == nil {
		o := f.getHttpOps()
		aciFile, cd, err := o.DownloadImage(u)
		if err != nil {
			return nil, nil, err
		}
		return aciFile, cd, nil
	}

	return f.fetchVerifiedURL(app, u, a)
}

func (f *nameFetcher) fetchVerifiedURL(app *discovery.App, u *url.URL, a *asc) (readSeekCloser, *cacheData, error) {
	appName := app.Name.String()
	f.maybeFetchPubKeys(appName)

	o := f.getHttpOps()
	ascFile, retry, err := o.DownloadSignature(a)
	if err != nil {
		return nil, nil, err
	}
	defer func() { maybeClose(ascFile) }()

	if !retry {
		if err := f.checkIdentity(appName, ascFile); err != nil {
			return nil, nil, err
		}
	}

	aciFile, cd, err := o.DownloadImage(u)
	if err != nil {
		return nil, nil, err
	}
	defer func() { maybeClose(aciFile) }()

	if retry {
		ascFile, err = o.DownloadSignatureAgain(a)
		if err != nil {
			return nil, nil, err
		}
	}

	if err := f.validate(appName, aciFile, ascFile); err != nil {
		return nil, nil, err
	}
	retAciFile := aciFile
	aciFile = nil
	return retAciFile, cd, nil
}

func (f *nameFetcher) maybeFetchPubKeys(appName string) {
	if f.TrustKeysFromHttps && !f.InsecureFlags.SkipTlsCheck() {
		m := &pubkey.Manager{
			AuthPerHost:        f.Headers,
			InsecureAllowHttp:  false,
			TrustKeysFromHttps: true,
			Ks:                 f.Ks,
			Debug:              f.Debug,
		}
		pkls, err := m.GetPubKeyLocations(appName)
		// We do not bail out here, because if fetching the
		// public keys fails but we already trust the key, we
		// should be able to run the image anyway.
		if err != nil {
			stderr("error determining key location: %v", err)
		} else {
			if err := m.AddKeys(pkls, appName, pubkey.AcceptForce, pubkey.OverrideDeny); err != nil {
				stderr("error adding keys: %v", err)
			}
		}
	}
}

func (f *nameFetcher) checkIdentity(appName string, ascFile io.ReadSeeker) error {
	if _, err := ascFile.Seek(0, 0); err != nil {
		return fmt.Errorf("error seeking signature file: %v", err)
	}
	empty := bytes.NewReader([]byte{})
	if _, err := f.Ks.CheckSignature(appName, empty, ascFile); err != nil {
		if _, ok := err.(pgperrors.SignatureError); !ok {
			return err
		}
	}
	return nil
}

func (f *nameFetcher) validate(appName string, aciFile, ascFile io.ReadSeeker) error {
	v, err := newValidator(aciFile)
	if err != nil {
		return err
	}

	if err := v.ValidateName(appName); err != nil {
		return err
	}

	entity, err := v.ValidateWithSignature(f.Ks, ascFile)
	if err != nil {
		return err
	}

	if _, err := aciFile.Seek(0, 0); err != nil {
		return fmt.Errorf("error seeking ACI file: %v", err)
	}

	printIdentities(entity)
	return nil
}

func (f *nameFetcher) maybeOverrideAscFetcherWithRemote(ascURL string, a *asc) {
	if a.Fetcher != nil {
		return
	}
	a.Location = ascURL
	a.Fetcher = f.getHttpOps().GetAscRemoteFetcher()
}

func (f *nameFetcher) getHttpOps() *httpOps {
	return &httpOps{
		InsecureSkipTLSVerify: f.InsecureFlags.SkipTlsCheck(),
		S:       f.S,
		Headers: f.Headers,
	}
}
