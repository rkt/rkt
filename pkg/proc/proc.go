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

package proc

import (
	"bufio"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var (
	ErrNotRoot = errors.New("must be root to access all process information")

	// socket symlinks in /proc/<pid>/fds take the form:
	// 	type:[inode]
	sre = regexp.MustCompile(`^([[:alpha:]]+):\[([[:digit:]]+)\]$`)
	// we hardcode the group indices because golang doesn't make named groups any easier :<
	sreTypei  = 1
	sreInodei = 2
)

const (
	procfs    = "/proc/"
	unixSocks = "/proc/net/unix"
)

// LiveProcs is similar to `man 1 fuser`; it takes a prefix and returns a map
// of PIDs of any processes accessing files with the prefix.
// A process is considered to be accessing a file if it has an open file
// descriptor directly referencing the file, has an open Unix socket
// referencing a file, or has a file mapped into memory.
// This operation is inherently racy (both false positives and false negatives
// are possible) and hence this should be considered an approximation only.
// TODO(jonboulle): map filename(string) -> []int(pids) instead
func LiveProcs(prefix string) (map[int][]string, error) {
	if os.Getegid() != 0 {
		return nil, ErrNotRoot
	}
	skts, err := unixSocketsWithPrefix(prefix)
	if err != nil {
		return nil, err
	}
	ps, err := ioutil.ReadDir(procfs)
	if err != nil {
		return nil, err
	}
	pids := make(map[int][]string)
	self := os.Getpid()
	for _, p := range ps {
		pid, err := strconv.Atoi(p.Name())
		if err != nil {
			continue
		}
		if pid == self {
			continue
		}

		// Parse file descriptors
		pdir := filepath.Join(procfs, p.Name(), "fd")
		fds, err := ioutil.ReadDir(pdir)
		switch {
		case err == nil:
		case os.IsNotExist(err):
			// assume we're too late
			continue
		default:
			return nil, err
		}
		links := make([]string, len(fds))
		for i, fd := range fds {
			links[i] = path.Join(pdir, fd.Name())
		}

		for _, path := range fdsWithPrefix(links, prefix, skts) {
			if _, ok := pids[pid]; ok {
				pids[pid] = make([]string, 0)
			}
			pids[pid] = append(pids[pid], path)
		}

		// Parse maps
		mfile := filepath.Join(procfs, p.Name(), "maps")
		mfh, err := os.Open(mfile)
		switch {
		case err == nil:
		case os.IsNotExist(err):
			// assume we're too late
			continue
		default:
			return nil, err
		}
		paths, err := mmapsWithPrefix(mfh, prefix)
		mfh.Close()
		if err != nil {
			return nil, err
		}
		for _, path := range paths {
			if _, ok := pids[pid]; ok {
				pids[pid] = make([]string, 0)
			}
			pids[pid] = append(pids[pid], path)
		}

	}
	return pids, nil
}

// unixSocketsWithPrefix returns a map (of inode->path) describing the Unix
// sockets open on the system accessing any paths with the given prefix
func unixSocketsWithPrefix(pre string) (map[string]string, error) {
	fh, err := os.Open(unixSocks)
	if err != nil {
		return nil, err
	}
	defer fh.Close()
	socks := make(map[string]string)
	scanner := bufio.NewScanner(fh)
	for scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return nil, err
		}
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) != 8 {
			continue
		}
		// lines are of the form:
		//    Num RefCount Protocol Flags Type St Inode Path
		inode := parts[6]
		pathname := parts[7]
		pathname = strings.TrimPrefix(pathname, "@")
		if strings.HasPrefix(pathname, pre) {
			socks[inode] = pathname
		}
	}
	return socks, nil
}

// fdsWithPrefix takes a list of fds (which should represent absolute paths to
// file descriptor links, i.e. as in /proc/<pid>/fd/), determines which files
// the fds represent, and returns a subset of those files that have the given
// prefix.
func fdsWithPrefix(fds []string, pre string, socks map[string]string) []string {
	var paths []string
	for _, fd := range fds {
		dest, err := os.Readlink(fd)
		if err != nil {
			continue
		}
		// Simple case: fd directly references a file we're interested in
		if strings.HasPrefix(dest, pre) {
			paths = append(paths, dest)
		}
		// Check for Unix sockets
		if m := sre.FindStringSubmatch(dest); m != nil {
			if m[sreTypei] != "socket" {
				continue
			}
			if p, ok := socks[m[sreInodei]]; ok {
				paths = append(paths, p)
			}
		}
	}
	return paths
}

// mmapsWithPrefix takes a Reader which should represent a file in the
// /proc/*/maps format (`man 5 proc`), and returns a list of paths in the map
// that have the given prefix
func mmapsWithPrefix(r io.Reader, pre string) ([]string, error) {
	var paths []string
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return nil, err
		}
		line := scanner.Text()
		parts := strings.Fields(line)
		// Lines are of the form:
		//    address perms offset dev inode
		//    address perms offset dev inode pathname
		if len(parts) < 6 {
			continue
		}
		pathname := parts[5]
		if strings.HasPrefix(pathname, pre) {
			paths = append(paths, pathname)
		}
	}
	return paths, nil
}
