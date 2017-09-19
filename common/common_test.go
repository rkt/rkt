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

package common

import (
	"testing"

	"github.com/appc/spec/schema/types"
)

func TestImageNameToAppName(t *testing.T) {
	for _, tt := range []struct {
		in  types.ACIdentifier
		out types.ACName
		err bool
	}{
		{
			in:  types.ACIdentifier("coreos.com/etcd:v2.0.0"),
			out: types.ACName("etcd-v2-0-0"),
			err: false,
		},
		{
			in:  types.ACIdentifier("coreos.com/etcd"),
			out: types.ACName("etcd"),
			err: false,
		},
		{
			in:  types.ACIdentifier("docker://registry.hub.docker.com/library/fedora"),
			out: types.ACName("fedora"),
			err: false,
		},
	} {
		appName, err := ImageNameToAppName(tt.in)
		if err != nil {
			if !tt.err {
				t.Fatal(err)
			}
		} else if appName == nil {
			t.Errorf("got nil app name without any error")
		} else if *appName != tt.out {
			t.Errorf("got %s app name, expected %s app name", appName, tt.out)
		}
	}
}
