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
	"github.com/coreos/rkt/Godeps/_workspace/src/github.com/appc/spec/schema"
	"github.com/coreos/rkt/Godeps/_workspace/src/github.com/appc/spec/schema/types"
)

// findResources find value of last isolator for particular type.
func findResources(isolators types.Isolators) (mem, cpus int64) {
	for _, i := range isolators {
		switch v := i.Value().(type) {
		case *types.ResourceMemory:
			memQuantity := v.Limit()
			mem = memQuantity.Value()
		case *types.ResourceCPU:
			cpusQuantity := v.Limit()
			cpus = cpusQuantity.Value()
		}
	}

	return mem, cpus
}

// GetAppsResources return values specfied by user in pod-manifest.
// Function expects a podmanifest apps.
// Return aggregate quantity of mem[B] and cpus.
func GetAppsResources(apps schema.AppList) (total_cpus, total_mem int64) {
	for i := range apps {
		ra := &apps[i]
		app := ra.App
		mem, cpus := findResources(app.Isolators)
		total_cpus += cpus
		total_mem += mem
	}
	return total_cpus, total_mem
}
