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
	"fmt"
	"io/ioutil"
	"os"

	"github.com/appc/spec/schema"
	"github.com/rkt/rkt/api/v1"
	"github.com/rkt/rkt/common"
	pkgPod "github.com/rkt/rkt/pkg/pod"
)

// appStateFunc fills in known state information:
// * App.State
// * App.CreatedAt
// * App.StartedAt
// * App.FinishedAt
// * App.ExitCode
type appStateFunc func(*v1.App, *pkgPod.Pod) error

// AppsForPod returns the apps of the pod with the given uuid in the given data directory.
// If appName is non-empty, then only the app with the given name will be returned.
func AppsForPod(uuid, dataDir string, appName string) ([]*v1.App, error) {
	p, err := pkgPod.PodFromUUIDString(dataDir, uuid)
	if err != nil {
		return nil, err
	}
	defer p.Close()

	return appsForPod(p, appName, appStateInMutablePod)
}

func appsForPod(p *pkgPod.Pod, appName string, appState appStateFunc) ([]*v1.App, error) {
	_, podManifest, err := p.PodManifest()
	if err != nil {
		return nil, err
	}

	var apps []*v1.App
	for _, ra := range podManifest.Apps {
		if appName != "" && appName != ra.Name.String() {
			continue
		}

		app, err := newApp(&ra, podManifest, p, appState)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Cannot get app status: %v", err)
			continue
		}

		apps = append(apps, app)
	}

	return apps, nil
}

// newApp constructs the App object with the runtime app and pod manifest.
func newApp(ra *schema.RuntimeApp, podManifest *schema.PodManifest, pod *pkgPod.Pod, appState appStateFunc) (*v1.App, error) {
	app := &v1.App{
		Name:            ra.Name.String(),
		ImageID:         ra.Image.ID.String(),
		UserAnnotations: ra.App.UserAnnotations,
		UserLabels:      ra.App.UserLabels,
	}

	for _, mnt := range ra.Mounts {
		app.Mounts = append(app.Mounts, &v1.Mount{
			Name:          mnt.Volume.String(),
			ContainerPath: mnt.Path,
			HostPath:      mnt.AppVolume.Source,
			ReadOnly:      *mnt.AppVolume.ReadOnly,
		})
	}

	// Generate state.
	if err := appState(app, pod); err != nil {
		return nil, fmt.Errorf("error getting app's state: %v", err)
	}

	return app, nil
}

func appStateInMutablePod(app *v1.App, pod *pkgPod.Pod) error {
	app.State = v1.AppStateUnknown

	defer func() {
		if pod.IsAfterRun() {
			// If the pod is hard killed, set the app to 'exited' state.
			// Other than this case, status file is guaranteed to be written.
			if app.State != v1.AppStateExited {
				app.State = v1.AppStateExited
				t, err := pod.GCMarkedTime()
				if err != nil {
					fmt.Fprintf(os.Stderr, "Cannot get GC marked time: %v", err)
				}
				if !t.IsZero() {
					finishedAt := t.UnixNano()
					app.FinishedAt = &finishedAt
				}
			}
		}
	}()

	// Check if the app is created.
	fi, err := os.Stat(common.AppCreatedPath(pod.Path(), app.Name))
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("cannot stat app creation file: %v", err)
		}
		return nil
	}

	app.State = v1.AppStateCreated
	createdAt := fi.ModTime().UnixNano()
	app.CreatedAt = &createdAt

	// Check if the app is started.
	fi, err = os.Stat(common.AppStartedPath(pod.Path(), app.Name))
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("cannot stat app started file: %v", err)
		}
		return nil
	}

	app.State = v1.AppStateRunning
	startedAt := fi.ModTime().UnixNano()
	app.StartedAt = &startedAt

	// Check if the app is exited.
	appStatusFile := common.AppStatusPath(pod.Path(), app.Name)
	fi, err = os.Stat(appStatusFile)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("cannot stat app exited file: %v", err)
		}
		return nil
	}

	app.State = v1.AppStateExited
	finishedAt := fi.ModTime().UnixNano()
	app.FinishedAt = &finishedAt

	// Read exit code.
	exitCode, err := readExitCode(appStatusFile)
	if err != nil {
		return err
	}
	app.ExitCode = &exitCode

	return nil
}

// appStateInImmutablePod infers most App state from the Pod itself, since all apps are created and destroyed with the Pod
func appStateInImmutablePod(app *v1.App, pod *pkgPod.Pod) error {
	app.State = appStateFromPod(pod)

	t, err := pod.CreationTime()
	if err != nil {
		return err
	}
	createdAt := t.UnixNano()
	app.CreatedAt = &createdAt

	code, err := pod.AppExitCode(app.Name)
	if err == nil {
		// there is an exit code, it is definitely Exited
		app.State = v1.AppStateExited
		exitCode := int32(code)
		app.ExitCode = &exitCode
	}

	start, err := pod.StartTime()
	if err != nil {
		return err
	}
	if !start.IsZero() {
		startedAt := start.UnixNano()
		app.StartedAt = &startedAt
	}
	// the best we can guess for immutable pods
	finish, err := pod.GCMarkedTime()
	if err != nil {
		return err
	}
	if !finish.IsZero() {
		finishedAt := finish.UnixNano()
		app.FinishedAt = &finishedAt
	}

	return nil
}

func appStateFromPod(pod *pkgPod.Pod) v1.AppState {
	switch pod.State() {
	case pkgPod.Embryo, pkgPod.Preparing, pkgPod.AbortedPrepare:
		return v1.AppStateUnknown
	case pkgPod.Prepared:
		return v1.AppStateCreated
	case pkgPod.Running:
		return v1.AppStateRunning
	case pkgPod.Deleting, pkgPod.ExitedDeleting, pkgPod.Exited, pkgPod.ExitedGarbage, pkgPod.Garbage:
		return v1.AppStateExited
	default:
		return v1.AppStateUnknown
	}
}

func readExitCode(path string) (int32, error) {
	var exitCode int32

	b, err := ioutil.ReadFile(path)
	if err != nil {
		return -1, fmt.Errorf("cannot read app exited file: %v", err)
	}
	if _, err := fmt.Sscanf(string(b), "%d", &exitCode); err != nil {
		return -1, fmt.Errorf("cannot parse exit code: %v", err)
	}
	return exitCode, nil
}
