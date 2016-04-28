// Copyright 2016 The rkt Authors
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

// +build kvm

package main

import "testing"

func TestCaps(t *testing.T) {
	// KVM is running VMs as stage1 pods, so root has access to all VM options.
	// The case with access to PID 1 is skipped...
	// KVM flavor runs systemd stage1 with full capabilities in stage1 (pid=1)
	// so expect every capability enabled
	NewCapsTest(true, []int{2}).Execute(t)
}
