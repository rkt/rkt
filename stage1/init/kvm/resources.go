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

package kvm

import (
	"runtime"

	"github.com/coreos/rkt/Godeps/_workspace/src/github.com/appc/spec/schema"
	"github.com/coreos/rkt/Godeps/_workspace/src/github.com/appc/spec/schema/types"
)

//By default app would get 128MB
const default_mem int64 = 128

// findResources finds value of last isolator for particular type.
func findResources(isolators types.Isolators) (mem, cpus int64) {
	for _, i := range isolators {
		switch v := i.Value().(type) {
		case *types.ResourceMemory:
			memQuantity := v.Limit()
			mem = memQuantity.Value()
			// Convert bytes into megabytes
			mem /= 1024 * 1024
		case *types.ResourceCPU:
			cpusQuantity := v.Limit()
			cpus = cpusQuantity.Value()
		}
	}
	if mem == 0 {
		mem = default_mem
	}

	return mem, cpus
}

// GetAppsResources returns values specfied by user in pod-manifest.
// Function expects a podmanifest apps.
// Return aggregate quantity of mem (in MB) and cpus.
func GetAppsResources(apps schema.AppList) (totalCpus, totalMem int64) {
	foundUnspecifiedCpu := false
	for i := range apps {
		ra := &apps[i]
		app := ra.App
		mem, cpus := findResources(app.Isolators)
		if cpus == 0 {
			foundUnspecifiedCpu = true
		}
		totalCpus += cpus
		totalMem += mem
	}
	// If user doesn't specify cpus for at least one app, we set no limit for
	// whole pod.
	if foundUnspecifiedCpu {
		totalCpus = int64(runtime.NumCPU())
	}

	// Increase amount of memory by 128MB for system.
	totalMem += 128

	return totalCpus, totalMem
}
