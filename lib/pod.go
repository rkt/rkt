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

package rkt

import (
	"github.com/rkt/rkt/api/v1"
	pkgPod "github.com/rkt/rkt/pkg/pod"
)

// NewPodFromInternalPod converts *pkgPod.Pod to *Pod
func NewPodFromInternalPod(p *pkgPod.Pod) (*v1.Pod, error) {
	pod := &v1.Pod{
		UUID:     p.UUID.String(),
		State:    p.State(),
		Networks: p.Nets,
	}

	startTime, err := p.StartTime()
	if err != nil {
		return nil, err
	}

	if !startTime.IsZero() {
		startedAt := startTime.Unix()
		pod.StartedAt = &startedAt
	}

	if !p.PodManifestAvailable() {
		return pod, nil
	}
	// TODO(vc): we should really hold a shared lock here to prevent gc of the pod
	_, manifest, err := p.PodManifest()
	if err != nil {
		return nil, err
	}

	for _, app := range manifest.Apps {
		// for backwards compatibility
		pod.AppNames = append(pod.AppNames, app.Name.String())
	}

	var appState appStateFunc
	if p.IsMutable() {
		appState = appStateInMutablePod
	} else {
		appState = appStateInImmutablePod
	}
	pod.Apps, err = appsForPod(p, "", appState)
	if err != nil {
		return nil, err
	}

	if len(manifest.UserAnnotations) > 0 {
		pod.UserAnnotations = make(map[string]string)
		for name, value := range manifest.UserAnnotations {
			pod.UserAnnotations[name] = value
		}
	}

	if len(manifest.UserLabels) > 0 {
		pod.UserLabels = make(map[string]string)
		for name, value := range manifest.UserLabels {
			pod.UserLabels[name] = value
		}
	}

	return pod, nil
}
