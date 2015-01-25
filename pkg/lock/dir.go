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

// Package lock implements simple locking primitives on a
// directory using flock
package lock

import (
	"errors"
	"syscall"
)

var (
	ErrLocked     = errors.New("directory already locked")
	ErrNotExist   = errors.New("directory does not exist")
	ErrPermission = errors.New("permission denied")
)

// DirLock represents a lock on a directory
type DirLock struct {
	dir string
	fd  int
}

// TryExclusiveLock takes an exclusive lock on a directory without blocking.
// This is idempotent when the DirLock already represents an exclusive lock,
// and tries promote a shared lock to exclusive atomically.
// It will return ErrLocked if any lock is already held on the directory.
func (l *DirLock) TryExclusiveLock() error {
	err := syscall.Flock(l.fd, syscall.LOCK_EX|syscall.LOCK_NB)
	if err == syscall.EWOULDBLOCK {
		err = ErrLocked
	}
	return err
}

// TryExclusiveLock takes an exclusive lock on a directory without blocking.
// It will return ErrLocked if any lock is already held on the directory.
func TryExclusiveLock(dir string) (*DirLock, error) {
	l, err := NewLock(dir)
	if err != nil {
		return nil, err
	}
	err = l.TryExclusiveLock()
	if err != nil {
		return nil, err
	}
	return l, err
}

// ExclusiveLock takes an exclusive lock on a directory.
// This is idempotent when the DirLock already represents an exclusive lock,
// and promotes a shared lock to exclusive atomically.
// It will block if an exclusive lock is already held on the directory.
func (l *DirLock) ExclusiveLock() error {
	return syscall.Flock(l.fd, syscall.LOCK_EX)
}

// ExclusiveLock takes an exclusive lock on a directory.
// It will block if an exclusive lock is already held on the directory.
func ExclusiveLock(dir string) (*DirLock, error) {
	l, err := NewLock(dir)
	if err == nil {
		err = l.ExclusiveLock()
	}
	if err != nil {
		return nil, err
	}
	return l, nil
}

// TrySharedLock takes a co-operative (shared) lock on a directory without blocking.
// This is idempotent when the DirLock already represents a shared lock,
// and tries demote an exclusive lock to shared atomically.
// It will return ErrLocked if an exclusive lock already exists on the directory.
func (l *DirLock) TrySharedLock() error {
	err := syscall.Flock(l.fd, syscall.LOCK_SH|syscall.LOCK_NB)
	if err == syscall.EWOULDBLOCK {
		err = ErrLocked
	}
	return err
}

// TrySharedLock takes a co-operative (shared) lock on a directory without blocking.
// It will return ErrLocked if an exclusive lock already exists on the directory.
func TrySharedLock(dir string) (*DirLock, error) {
	l, err := NewLock(dir)
	if err != nil {
		return nil, err
	}
	err = l.TrySharedLock()
	if err != nil {
		return nil, err
	}
	return l, nil
}

// SharedLock takes a co-operative (shared) lock on a directory.
// This is idempotent when the DirLock already represents a shared lock,
// and demotes an exclusive lock to shared atomically.
// It will block if an exclusive lock is already held on the directory.
func (l *DirLock) SharedLock() error {
	return syscall.Flock(l.fd, syscall.LOCK_SH)
}

// SharedLock takes a co-operative (shared) lock on a directory.
// It will block if an exclusive lock is already held on the directory.
func SharedLock(dir string) (*DirLock, error) {
	l, err := NewLock(dir)
	if err != nil {
		return nil, err
	}
	err = l.SharedLock()
	if err != nil {
		return nil, err
	}
	return l, nil
}

// Unlock unlocks the lock
func (l *DirLock) Unlock() error {
	return syscall.Flock(l.fd, syscall.LOCK_UN)
}

// Fd returns the lock's file descriptor, or an error if the lock is closed
func (l *DirLock) Fd() (int, error) {
	var err error
	if l.fd == -1 {
		err = errors.New("lock closed")
	}
	return l.fd, err
}

// Close closes the lock which implicitly unlocks it as well
func (l *DirLock) Close() error {
	fd := l.fd
	l.fd = -1
	return syscall.Close(fd)
}

// NewLock opens a new lock on a directory without acquisition
func NewLock(dir string) (*DirLock, error) {
	l := &DirLock{dir: dir, fd: -1}

	// we can't use os.OpenFile as Go sets O_CLOEXEC
	lfd, err := syscall.Open(l.dir, syscall.O_RDONLY|syscall.O_DIRECTORY, 0)
	if err != nil {
		if err == syscall.ENOENT {
			err = ErrNotExist
		} else if err == syscall.EACCES {
			err = ErrPermission
		}
		return nil, err
	}
	l.fd = lfd

	return l, nil
}
