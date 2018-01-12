// Copyright 2017 The rkt Authors
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
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
)

const (
	progName = "mds-example"
)

var (
	signCommand   *flag.FlagSet
	verifyCommand *flag.FlagSet

	signFileFlag      *string
	signSignatureFlag *string

	verifyFileFlag      *string
	verifyPodUUID       *string
	verifySignatureFlag *string
)

func init() {
	signCommand = flag.NewFlagSet("sign", flag.ExitOnError)
	signFileFlag = signCommand.String("file", "", "File to sign")
	signSignatureFlag = signCommand.String("signature", "", "Signature output file")

	verifyCommand = flag.NewFlagSet("verify", flag.ExitOnError)
	verifyFileFlag = verifyCommand.String("file", "", "File to verify")
	verifyPodUUID = verifyCommand.String("uuid", "", "UUID of the sender pod")
	verifySignatureFlag = verifyCommand.String("signature", "", "Signature file")
}

func usage() {
	fmt.Fprintf(os.Stderr, "USAGE:\n")
	fmt.Fprintf(os.Stderr, "\t%s <command> <args>\n", progName)
	fmt.Fprintf(os.Stderr, "\nCOMMANDS:\n")
	fmt.Fprintf(os.Stderr, "\tsign --file=FILE --signature=SIGNATURE_FILE\n")
	fmt.Fprintf(os.Stderr, "\t\tsign a file with the metadata server and save signature to a file\n")
	fmt.Fprintf(os.Stderr, "\tverify --file=FILE --uuid=POD_UUID --signature=SIGNATURE_FILE\n")
	fmt.Fprintf(os.Stderr, "\t\tverify a file signed with the metadata server\n")
}

func parseArgs(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("missing command")
	}
	switch args[1] {
	case "sign":
		signCommand.Parse(args[2:])
	case "verify":
		verifyCommand.Parse(args[2:])
	default:
		return fmt.Errorf("%q is not a valid command.", args[1])
	}

	return nil
}

func sign(mdsURL, p, outPath string) error {
	signPath := path.Join("/", "acMetadata", "v1", "pod", "hmac", "sign")
	signURL := mdsURL + signPath

	content, err := ioutil.ReadFile(p)
	if err != nil {
		return fmt.Errorf("error reading input file: %v", err)
	}

	v := url.Values{}
	v.Set("content", string(content))

	rsp, err := http.PostForm(signURL, v)
	if err != nil {
		return fmt.Errorf("error generating request: %v", err)
	}
	defer rsp.Body.Close()

	data, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		return fmt.Errorf("error reading response: %v", err)
	}

	if rsp.StatusCode != http.StatusOK {
		return fmt.Errorf("error signing: %v", string(data))
	}

	if err := ioutil.WriteFile(outPath, data, 0600); err != nil {
		return fmt.Errorf("error writing signature: %v", err)
	}

	return nil
}

func verify(mdsURL, p, uuid, signaturePath string) (bool, error) {
	signPath := path.Join("/", "acMetadata", "v1", "pod", "hmac", "verify")
	signURL := mdsURL + signPath

	content, err := ioutil.ReadFile(p)
	if err != nil {
		return false, fmt.Errorf("error reading input file: %v", err)
	}

	signatureBytes, err := ioutil.ReadFile(signaturePath)
	if err != nil {
		return false, fmt.Errorf("error reading signature file: %v", err)
	}

	v := url.Values{}
	v.Set("content", string(content))
	v.Set("uuid", uuid)
	v.Set("signature", string(signatureBytes))

	rsp, err := http.PostForm(signURL, v)
	if err != nil {
		return false, fmt.Errorf("error generating request: %v", err)
	}
	defer rsp.Body.Close()

	switch rsp.StatusCode {
	case http.StatusOK:
		return true, nil
	case http.StatusForbidden:
		return false, nil
	default:
		return false, fmt.Errorf("unknown error: %v", http.StatusText(rsp.StatusCode))
	}
}

func main() {
	if err := parseArgs(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", progName, err)
		usage()
		os.Exit(1)
	}

	mdsURL := os.Getenv("AC_METADATA_URL")
	if mdsURL == "" {
		fmt.Fprintf(os.Stderr, "%s: $AC_METADATA_URL env variable not found, are you in a rkt container and is the rkt metadata service running on the host?\n", progName)
		os.Exit(1)
	}

	if signCommand.Parsed() {
		if *signFileFlag == "" || *signSignatureFlag == "" {
			fmt.Fprintf(os.Stderr, "%s: 'sign' needs a file to sign and an output file\n", progName)
			usage()
			os.Exit(1)
		}

		if err := sign(mdsURL, *signFileFlag, *signSignatureFlag); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", progName, err)
			os.Exit(1)
		}
	} else if verifyCommand.Parsed() {
		if *verifyFileFlag == "" || *verifySignatureFlag == "" || *verifyPodUUID == "" {
			fmt.Fprintf(os.Stderr, "%s: 'verify' needs a file to verify, the pod UUID of the sender, and a signature file\n", progName)
			usage()
			os.Exit(1)
		}

		ok, err := verify(mdsURL, *verifyFileFlag, *verifyPodUUID, *verifySignatureFlag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", progName, err)
			os.Exit(1)
		}

		if ok {
			fmt.Println("signature OK")
		} else {
			fmt.Println("signature INVALID")
			os.Exit(1)
		}
	}

	os.Exit(0)
}
