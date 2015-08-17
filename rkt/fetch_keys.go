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
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/coreos/rkt/Godeps/_workspace/src/github.com/appc/spec/discovery"
	"github.com/coreos/rkt/Godeps/_workspace/src/golang.org/x/crypto/openpgp"
)

// getPubKeyLocations discovers one location at prefix
func getPubKeyLocations(prefix string, allowHTTP bool, debug bool) ([]string, error) {
	if prefix == "" {
		return nil, fmt.Errorf("empty prefix")
	}

	kls, err := metaDiscoverPubKeyLocations(prefix, allowHTTP, debug)
	if err != nil {
		return nil, fmt.Errorf("prefix meta discovery error: %v", err)
	}

	if len(kls) == 0 {
		return nil, fmt.Errorf("meta discovery on %s resulted in no keys", prefix)
	}

	return kls, nil
}

// metaDiscoverPubKeyLocations discovers the public key through ACDiscovery by applying prefix as an ACApp
func metaDiscoverPubKeyLocations(prefix string, allowHTTP bool, debug bool) ([]string, error) {
	app, err := discovery.NewAppFromString(prefix)
	if err != nil {
		return nil, err
	}

	ep, attempts, err := discovery.DiscoverPublicKeys(*app, allowHTTP)
	if err != nil {
		return nil, err
	}

	if debug {
		for _, a := range attempts {
			stderr("meta tag 'ac-discovery-pubkeys' not found on %s: %v", a.Prefix, a.Error)
		}
	}

	return ep.Keys, nil
}

// getPubKey retrieves a public key (if remote), and verifies it's a gpg key
func getPubKey(scheme, location string, allowHTTP bool) (*os.File, error) {
	switch scheme {
	case "":
		return os.Open(location)
	case "http":
		if !allowHTTP {
			return nil, fmt.Errorf("--insecure-allow-http required for http URLs")
		}
		fallthrough
	case "https":
		return downloadKey(location)
	}

	return nil, fmt.Errorf("only http and https urls supported")
}

// downloadKey retrieves the file, storing it in a deleted tempfile
func downloadKey(url string) (*os.File, error) {
	tf, err := ioutil.TempFile("", "")
	if err != nil {
		return nil, fmt.Errorf("error creating tempfile: %v", err)
	}
	os.Remove(tf.Name()) // no need to keep the tempfile around

	defer func() {
		if err != nil {
			tf.Close()
		}
	}()

	res, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error getting key: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad HTTP status code: %d", res.StatusCode)
	}

	if _, err := io.Copy(tf, res.Body); err != nil {
		return nil, fmt.Errorf("error copying key: %v", err)
	}

	if _, err = tf.Seek(0, os.SEEK_SET); err != nil {
		return nil, fmt.Errorf("error seeking: %v", err)
	}

	return tf, nil
}

// addKeys adds the keys listed in pkls at prefix
func addKeys(pkls []string, prefix string, allowHTTP, forceAccept, allowOverride bool) error {
	ks := getKeystore()
	if ks == nil {
		panic("could not get the key store")
	}

	for _, pkl := range pkls {
		u, err := url.Parse(pkl)
		if err != nil {
			return err
		}
		pk, err := getPubKey(u.Scheme, pkl, allowHTTP)
		if err != nil {
			return fmt.Errorf("error accessing the key %s: %v", pkl, err)
		}
		defer pk.Close()

		exists, err := ks.TrustedKeyPrefixExists(prefix, pk)
		if err != nil {
			return fmt.Errorf("error reading the key %s: %v", pkl, err)
		}
		err = displayKey(prefix, pkl, pk)
		if err != nil {
			return fmt.Errorf("error displaying the key %s: %v", pkl, err)
		}
		if exists && !allowOverride {
			stderr("Key %q already in the keystore", pkl)
			continue
		}

		if globalFlags.TrustKeysFromHttps && u.Scheme == "https" {
			forceAccept = true
		}

		if !forceAccept {
			accepted, err := reviewKey()
			if err != nil {
				return fmt.Errorf("error reviewing key: %v", err)
			}
			if !accepted {
				stderr("Not trusting %q", pkl)
				continue
			}
		}

		if forceAccept {
			stderr("Trusting %q for prefix %q without fingerprint review.", pkl, prefix)
		} else {
			stderr("Trusting %q for prefix %q after fingerprint review.", pkl, prefix)
		}

		if prefix == "" {
			path, err := ks.StoreTrustedKeyRoot(pk)
			if err != nil {
				return fmt.Errorf("Error adding root key: %v", err)
			}
			stderr("Added root key at %q", path)
		} else {
			path, err := ks.StoreTrustedKeyPrefix(prefix, pk)
			if err != nil {
				return fmt.Errorf("Error adding key for prefix %q: %v", prefix, err)
			}
			stderr("Added key for prefix %q at %q", prefix, path)
		}
	}
	return nil
}

// displayKey shows the key summary
func displayKey(prefix string, location string, key *os.File) error {
	defer key.Seek(0, os.SEEK_SET)

	kr, err := openpgp.ReadArmoredKeyRing(key)
	if err != nil {
		return fmt.Errorf("error reading key: %v", err)
	}

	stderr("prefix: %q\nkey: %q", prefix, location)
	for _, k := range kr {
		stderr("gpg key fingerprint is: %s", fingerToString(k.PrimaryKey.Fingerprint))
		for _, sk := range k.Subkeys {
			stderr("    subkey fingerprint: %s", fingerToString(sk.PublicKey.Fingerprint))
		}
		for n, _ := range k.Identities {
			stderr("\t%s", n)
		}
	}
	return nil
}

// reviewKey asks the user to accept the key
func reviewKey() (bool, error) {
	in := bufio.NewReader(os.Stdin)
	for {
		stderr("Are you sure you want to trust this key (yes/no)?")
		input, err := in.ReadString('\n')
		if err != nil {
			return false, fmt.Errorf("error reading input: %v", err)
		}
		switch input {
		case "yes\n":
			return true, nil
		case "no\n":
			return false, nil
		default:
			stderr("Please enter 'yes' or 'no'")
		}
	}
}

func fingerToString(fpr [20]byte) string {
	str := ""
	for i, b := range fpr {
		if i > 0 && i%2 == 0 {
			str += " "
			if i == 10 {
				str += " "
			}
		}
		str += strings.ToUpper(fmt.Sprintf("%.2x", b))
	}
	return str
}
