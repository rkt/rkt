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

package flag

import (
	"testing"
)

func TestSkipVerify(t *testing.T) {
	opt := "none"
	flags, err := NewSecFlags(opt)
	if err != nil {
		t.Fatalf("Unexpected error when creating SecFlags with %q: %v", opt, err)
	}
	sv := &skipVerify{
		flags: flags,
	}

	bt := "true"
	if err := sv.Set(bt); err != nil {
		t.Fatalf("Unexpected error when setting --insecure-skip-verify to %q: %v", bt, err)
	}
	if !flags.SkipAllSecurityChecks() {
		t.Errorf("Expected --insecure-skip-verify to skip all security checks")
	}

	bf := "false"
	if err := sv.Set(bf); err != nil {
		t.Fatalf("Unexpected error when setting --insecure-skip-verify to %q: %v", bf, err)
	}
	if flags.SkipAnySecurityChecks() {
		t.Errorf("Expected --insecure-skip-verify to skip all security checks")
	}
}
