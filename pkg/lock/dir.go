// Copyright 2014 CoreOS, Inc.
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

package lock

// Package lock implements simple locking primitives on a directory using flock
import (
	"errors"
	"syscall"
)

var (
	ErrLocked = errors.New("directory already locked")
)

// DirLock represents a Directory with an active Lock.
type DirLock interface {
	// Close() closes the file representing the lock, implicitly unlocking.
	Close() error
	// Fd() returns the fd number for the file representing the lock
	Fd() (int, error)
}

// TryExclusiveLock takes an exclusive lock on a directory without blocking.
// It will return ErrLocked if any lock is already held on the directory.
func TryExclusiveLock(dir string) (DirLock, error) {
	l, err := newLock(dir)
	if err != nil {
		return nil, err
	}
	err = syscall.Flock(l.fd, syscall.LOCK_EX|syscall.LOCK_NB)
	if err == syscall.EWOULDBLOCK {
		err = ErrLocked
	}
	if err != nil {
		return nil, err
	}
	return l, err
}

// ExclusiveLock takes an exclusive lock on a directory.
// It will block if an exclusive lock is already held on the directory.
func ExclusiveLock(dir string) (DirLock, error) {
	l, err := newLock(dir)
	if err == nil {
		err = syscall.Flock(l.fd, syscall.LOCK_EX)
	}
	if err != nil {
		return nil, err
	}
	return l, nil
}

// TrySharedLock takes a co-operative (shared) lock on a directory without blocking.
// It will return ErrLocked if an exclusive lock already exists on the directory.
func TrySharedLock(dir string) (DirLock, error) {
	l, err := newLock(dir)
	if err != nil {
		return nil, err
	}
	err = syscall.Flock(l.fd, syscall.LOCK_SH|syscall.LOCK_NB)
	if err == syscall.EWOULDBLOCK {
		err = ErrLocked
	}
	if err != nil {
		return nil, err
	}
	return l, nil
}

// SharedLock takes a co-operative (shared) lock on a directory.
// It will block if an exclusive lock is already held on the directory.
func SharedLock(dir string) (DirLock, error) {
	l, err := newLock(dir)
	if err != nil {
		return nil, err
	}
	err = syscall.Flock(l.fd, syscall.LOCK_SH)
	if err != nil {
		return nil, err
	}
	return l, nil
}

type lock struct {
	dir string
	fd  int
}

// Fd returns the lock's file descriptor
func (l *lock) Fd() (int, error) {
	var err error
	if l.fd == -1 {
		err = errors.New("lock closed")
	}
	return l.fd, err
}

// Close closes the lock which implicitly unlocks it as well
func (l *lock) Close() error {
	fd := l.fd
	l.fd = -1
	return syscall.Close(fd)
}

// NewLock opens a new lock on a directory without acquisition
func newLock(dir string) (*lock, error) {
	l := &lock{dir: dir, fd: -1}

	// we can't use os.OpenFile as Go sets O_CLOEXEC
	lfd, err := syscall.Open(l.dir, syscall.O_RDONLY|syscall.O_DIRECTORY, 0)
	if err != nil {
		return nil, err
	}
	l.fd = lfd

	return l, nil
}
