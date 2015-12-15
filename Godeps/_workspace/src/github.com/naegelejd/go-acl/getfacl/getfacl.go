package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"strconv"

	"github.com/coreos/rkt/Godeps/_workspace/src/github.com/naegelejd/go-acl"
	os2 "github.com/coreos/rkt/Godeps/_workspace/src/github.com/naegelejd/go-acl/os"
	"github.com/coreos/rkt/Godeps/_workspace/src/github.com/naegelejd/go-acl/os/group"
)

func main() {
	recursive := flag.Bool("recursive", false, "recurse into directories")
	omitheader := flag.Bool("omit-header", false, "omit header for each path")

	flag.Parse()
	if flag.NArg() < 1 {
		flag.Usage()
	}

	for i := 0; i < flag.NArg(); i++ {
		if err := getfacl(flag.Arg(i), *recursive, !*omitheader); err != nil {
			log.Fatal(err)
		}
	}
}

func getfacl(p string, recursive, header bool) error {
	a, err := acl.GetFileAccess(p)
	if err != nil {
		return fmt.Errorf("Failed to get ACL from %s (%s)", p, err)
	}

	uid, gid, err := os2.Owner(p)
	if err != nil {
		return fmt.Errorf("Failed to lookup owner and group (%s)", err)
	}
	user, err := user.LookupId(strconv.Itoa(int(uid)))
	if err != nil {
		return fmt.Errorf("Failed to lookup user (%s)", err)
	}
	group, err := group.LookupId(strconv.Itoa(int(gid)))
	if err != nil {
		return fmt.Errorf("Failed to lookup group (%s)", err)
	}

	if header {
		fmt.Printf("# file: %s\n# user: %s\n# group: %s\n", p, user.Username, group.Name)
	}
	fmt.Println(a)

	// Free ACL before recursing
	a.Free()

	if recursive {
		if err := recurse(p, header); err != nil {
			return err
		}
	}
	return nil
}

func recurse(p string, header bool) error {
	fi, err := os.Stat(p)
	if err != nil {
		return fmt.Errorf("Failed to stat path %s (%s)", p, err)
	}
	if fi.IsDir() {
		dirents, err := ioutil.ReadDir(p)
		if err != nil {
			return err
		}
		for _, child := range dirents {
			if err := getfacl(filepath.Join(p, child.Name()), true, header); err != nil {
				return err
			}
		}
	}
	return nil
}
