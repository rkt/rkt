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

package store

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/coreos/rkt/pkg/lock"
	"github.com/hashicorp/errwrap"

	_ "github.com/cznic/ql/driver"
)

const (
	DbFilename = "ql.db"
)

// dbLock is used to guarantee both thread-safety and process-safety
// for db access.
type dbLock struct {
	// This lock is to make that access to the ql db file being blocking
	// since ql use an internal locking that will not block and return an
	// error when a lock is already held.
	fl *lock.FileLock
	// This lock is to avoid concurrent access from multiple goroutines.
	sync.Mutex
}

func newDBLock(dirPath string) (*dbLock, error) {
	l, err := lock.NewLock(dirPath, lock.Dir)
	if err != nil {
		return nil, err
	}
	return &dbLock{fl: l}, nil
}

func (dl *dbLock) lock() error {
	dl.Lock()
	if err := dl.fl.ExclusiveLock(); err != nil {
		dl.Unlock()
		return err
	}
	return nil
}

func (dl *dbLock) unlock() {
	if err := dl.fl.Unlock(); err != nil {
		// TODO(sgotti) what'll happen when df.fl.Unlock fails? From
		// man 2 flock looks like it'll happen only when the underlying
		// fd has been closed (in this case the lock has been released
		// when the fd has been closed, assuming no dup fd due to
		// forking etc...).

		// If there're other cases where it fails without unlocking,
		// there's no simple way to handle them.
		// Possible solutions:
		// * panic (done here)
		// * try to close the lock (and related fd), panic if close
		// fails and create a new lock.
		//
		// Passing a specific error to the caller and let it recover
		// creating a new store instance is tricky because we
		// don't know the lock state and cannot be sure on how to clean
		// this store instance

		panic(fmt.Errorf("failed to unlock the db flock: %v", err))
	}
	dl.Unlock()
}

type DB struct {
	dbdir string
	dl    *dbLock
	sqldb *sql.DB
}

func NewDB(dbdir string) (*DB, error) {
	if err := os.MkdirAll(dbdir, defaultPathPerm); err != nil {
		return nil, err
	}

	dl, err := newDBLock(dbdir)
	if err != nil {
		return nil, err
	}

	return &DB{dbdir: dbdir, dl: dl}, nil
}

func (db *DB) Open() error {
	if err := db.dl.lock(); err != nil {
		return err
	}

	sqldb, err := sql.Open("ql", filepath.Join(db.dbdir, DbFilename))
	if err != nil {
		db.dl.unlock()
		return err
	}
	db.sqldb = sqldb

	return nil
}

func (db *DB) Close() error {
	if db.sqldb == nil {
		panic("cas db, Close called without an open sqldb")
	}

	if err := db.sqldb.Close(); err != nil {
		return errwrap.Wrap(errors.New("cas db close failed"), err)
	}
	db.sqldb = nil

	// Don't close the flock as it will be reused.
	db.dl.unlock()
	return nil
}

func (db *DB) Begin() (*sql.Tx, error) {
	return db.sqldb.Begin()
}

type txfunc func(*sql.Tx) error

// Do Opens the db, executes DoTx and then Closes the DB
func (db *DB) Do(fns ...txfunc) error {
	err := db.Open()
	if err != nil {
		return err
	}
	defer db.Close()

	return db.DoTx(fns...)
}

// DoTx executes the provided txfuncs inside a unique transaction.
// If one of the functions returns an error the whole transaction is rolled back.
func (db *DB) DoTx(fns ...txfunc) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	for _, fn := range fns {
		if err := fn(tx); err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}
