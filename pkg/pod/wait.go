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

package pod

import (
	"time"

	"golang.org/x/net/context"
)

// WaitFinished waits for a pod to (run and) finish.
// This method blocks indefinitely and refreshes the pod state.
// It is the caller's responsibility to determine the actual terminal state.
func (p *Pod) WaitFinished() error {
	// isExited implies isExitedGarbage.
	for !p.IsFinished() {
		// this blocks (waits) as long as the pod is locked, i.e. in pepare, run, exitedGarbage, garbage state
		if err := p.SharedLock(); err != nil {
			return err
		}

		// unlock immediately
		if err := p.Unlock(); err != nil {
			return err
		}

		if err := p.refreshState(); err != nil {
			return err
		}

		// if we're in the gap between preparing and running in a split prepare/run-prepared usage, take a nap
		if p.isPrepared {
			time.Sleep(time.Second)
		}
	}

	return nil
}

// WaitReady blocks until the pod is ready by polling the readiness state every 100 milliseconds
// or until the given context is cancelled. This method refreshes the pod state.
func (p *Pod) WaitReady(ctx context.Context) error {
	f := func() bool {
		if err := p.refreshState(); err != nil {
			return false
		}

		return p.IsSupervisorReady()
	}

	return retry(ctx, f, 100*time.Millisecond)
}

// retry calls function f indefinitely with the given delay between invocations
// until f returns true or the given context is cancelled.
// It returns immediately without delay in case function f immediately returns true.
func retry(ctx context.Context, f func() bool, delay time.Duration) error {
	if f() {
		return nil
	}

	ticker := time.NewTicker(delay)
	errChan := make(chan error)

	go func() {
		defer close(errChan)

		for {
			select {
			case <-ctx.Done():
				errChan <- ctx.Err()
				return
			case <-ticker.C:
				if f() {
					return
				}
			}
		}
	}()

	return <-errChan
}
