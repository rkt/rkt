package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/coreos/rkt/Godeps/_workspace/src/github.com/naegelejd/go-acl"
)

// Usage: setfacl [-bkndRLP] { -m|-M|-x|-X ... } file ...
// -m, --modify=acl        modify the current ACL(s) of file(s)
// -M, --modify-file=file  read ACL entries to modify from file
// -x, --remove=acl        remove entries from the ACL(s) of file(s)
// -X, --remove-file=file  read ACL entries to remove from file
// -b, --remove-all        remove all extended ACL entries
// -k, --remove-default    remove the default ACL
//     --set=acl           set the ACL of file(s), replacing the current ACL
//     --set-file=file     read ACL entries to set from file
//     --mask              do recalculate the effective rights mask
// -n, --no-mask           don't recalculate the effective rights mask
// -d, --default           operations apply to the default ACL
// -R, --recursive         recurse into subdirectories
// -L, --logical           logical walk, follow symbolic links
// -P, --physical          physical walk, do not follow symbolic links
//     --restore=file      restore ACLs (inverse of `getfacl -R')
//     --test              test mode (ACLs are not modified)
// -v, --version           print version and exit
// -h, --help              this help text

// AUTOMATICALLY CREATED ENTRIES
// Initially, files and directories contain only the three base ACL entries for the owner, the group, and others.
// There are some rules that need to be satisfied in order for an ACL to be valid:

// * The  three  base  entries  cannot  be removed. There must be exactly one entry of each of these base entry
//   types.

// * Whenever an ACL contains named user entries or named group objects, it  must  also  contain  an  effective
//   rights mask.

// * Whenever  an  ACL  contains  any  Default  ACL entries, the three Default ACL base entries (default owner,
//   default group, and default others) must also exist.

// * Whenever a Default ACL contains named user entries or named group objects, it must also contain a  default
//   effective rights mask.

// To  help the user ensure these rules, setfacl creates entries from existing entries under the following condi‐
// tions:

// * If an ACL contains named user or named group entries, and no mask entry exists, a  mask  entry  containing
//   the  same permissions as the group entry is created. Unless the -n option is given, the permissions of the
//   mask entry are further adjusted to include the union of all permissions affected by the mask  entry.  (See
//   the -n option description).

// * If a Default ACL entry is created, and the Default ACL contains no owner, owning group, or others entry, a
//   copy of the ACL owner, owning group, or others entry is added to the Default ACL.

// * If a Default ACL contains named user entries or named group entries, and no  mask  entry  exists,  a  mask
//   entry  containing  the  same  permissions as the default Default ACL's group entry is added. Unless the -n
//   option is given, the permissions of the mask entry are further adjusted to inclu de the union of all  per‐
//   missions affected by the mask entry. (See the -n option description).
var (
	recursive    bool
	ignoreLinks  bool
	calcMask     bool
	applyDefault bool
	dryRun       bool
)

type Mode int

const (
	setMode int = iota
	modMode
	delMode
)

type ACLSetter func(p string) error

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	set := flag.String("s", "", "set ACL(s) of file(s), replacing the current ACL")
	mod := flag.String("m", "", "modify current ACL(s) of file(s)")
	del := flag.String("x", "", "remove entries from ACL(s) of file(s)")
	delall := flag.Bool("b", false, "remove all extended ACL entries")
	deldef := flag.Bool("k", false, "remove the default ACL")
	// fromfile := flag.String("f", "", "read ACL entries to set from file")
	// restore := flag.String("g", "", "restore ACL (inverse of `getfacl -R`)")

	flag.BoolVar(&applyDefault, "d", false, "apply operations to the default ACL")
	flag.BoolVar(&recursive, "r", false, "recurse into subdirectories")
	flag.BoolVar(&ignoreLinks, "p", false, "do NOT follow symbolic links")
	flag.BoolVar(&calcMask, "mask", true, "recalculate the effective rights mask")
	// flag.BoolVar(&dryRun, "n", false, "print the resulting ACLs")

	// var text string
	// var mode Mode

	// Sources
	// command line
	// existing entries

	// Modes
	// set new ACL entries
	// modify existing ACL entries
	// delete existing ACL entries
	// delete every entry
	// delete default entries
	// test existing entries

	// Set: parse ACL, apply ACL to every file
	// Mod: parse ACL, "add" to ACL of every file
	// Del: parse ACL, "find" and "remove" matching ACL of every file

	flag.Parse()
	if flag.NArg() < 1 {
		flag.Usage()
	}

	// User must select exactly one mode
	if countModes(len(*set) > 0, len(*mod) > 0, len(*del) > 0, *delall, *deldef) != 1 {
		fmt.Println("Wrong: must choose one of s|m|x|b|k")
		flag.Usage()
	}

	var g func(*acl.ACL) ACLSetter
	var a *acl.ACL

	if *delall {
		a = acl.New()
		g = deleteAll
	} else if *deldef {
		a = acl.New()
		g = deleteDefault
	} else {
		var source string
		switch {
		case len(*set) > 0:
			source = *set
			g = setACL
		case len(*mod) > 0:
			source = *mod
			g = modACL
		case len(*del) > 0:
			source = *del
			g = delACL
		default:
			log.Fatal("Invalid mode. Contact author")
		}
		var err error
		a, err = acl.Parse(source)
		if err != nil {
			log.Fatal(err)
		}
	}

	defer a.Free()
	f := g(a)

	for i := 0; i < flag.NArg(); i++ {
		p := flag.Arg(i)
		if err := apply(f, p); err != nil {
			log.Fatal(err)
		}
	}
}

func countModes(modes ...bool) int {
	count := 0
	for _, m := range modes {
		if m {
			count++
		}
	}
	return count
}

func calculateMask(a *acl.ACL) error {
	if calcMask {
		if err := a.CalcMask(); err != nil {
			return err
		}
	}
	return nil
}

func setACL(a *acl.ACL) ACLSetter {
	return func(p string) error {
		var err error
		if err = calculateMask(a); err != nil {
			return err
		}
		if applyDefault {
			err = a.SetFileDefault(p)
		} else {
			log.Printf("Setting %s on %s\n", a, p)
			err = a.SetFileAccess(p)
		}
		return err
	}
}

func modACL(a *acl.ACL) ACLSetter {
	return func(p string) error {
		var x *acl.ACL
		var err error

		if applyDefault {
			x, err = acl.GetFileDefault(p)
		} else {
			x, err = acl.GetFileAccess(p)
		}
		if err != nil {
			return err
		}

		// copy each entry to existing ACL
		for e := a.FirstEntry(); e != nil; e = a.NextEntry() {
			if err := x.AddEntry(e); err != nil {
				log.Printf("Error copying entry (%s)\n", err)
			}
		}

		if err = calculateMask(x); err != nil {
			return err
		}
		if applyDefault {
			err = x.SetFileDefault(p)
		} else {
			err = x.SetFileAccess(p)
		}
		return err
	}
}

func delACL(a *acl.ACL) ACLSetter {
	return func(p string) error {
		var x *acl.ACL
		var err error

		if applyDefault {
			x, err = acl.GetFileDefault(p)
		} else {
			x, err = acl.GetFileAccess(p)
		}
		if err != nil {
			return err
		}

		// TODO: remove existing ACL matching specified ACL
		// for each entry in a, for each entry in x, if tag and qualifier match, remove from x
		for delEntry := a.FirstEntry(); delEntry != nil; delEntry = a.NextEntry() {
			delTag, err := delEntry.GetTag()
			if err != nil {
				continue
				return err
			}
			delQual, err := delEntry.GetQualifier()
			if err != nil {
				continue
			}
			for exEntry := x.FirstEntry(); exEntry != nil; exEntry = x.NextEntry() {
				exTag, err := exEntry.GetTag()
				if err != nil {
					continue
				}
				exQual, err := exEntry.GetQualifier()
				if err != nil {
					continue
				}
				if delTag == exTag && delQual == exQual {
					if err := x.DeleteEntry(delEntry); err != nil {
						return err
					}
				}
			}
		}

		if err = calculateMask(x); err != nil {
			return err
		}
		if applyDefault {
			err = x.SetFileDefault(p)
		} else {
			err = x.SetFileAccess(p)
		}
		return err
	}
}

func deleteAll(a *acl.ACL) ACLSetter {
	return func(p string) error {
		var err error
		if err = calculateMask(a); err != nil {
			return err
		}
		if err = a.SetFileAccess(p); err != nil {
			return err
		}
		if err = a.SetFileDefault(p); err != nil {
			return err
		}
		return nil
	}
}

func deleteDefault(a *acl.ACL) ACLSetter {
	return func(p string) error {
		var err error
		if err = calculateMask(a); err != nil {
			return err
		}
		if err = a.SetFileDefault(p); err != nil {
			return err
		}
		return nil
	}
}

func isLink(p string) (bool, error) {
	fi, err := os.Lstat(p)
	if err != nil {
		return false, err
	}
	if fi.Mode()&os.ModeSymlink == os.ModeSymlink {
		return true, nil
	}
	return false, nil
}

func apply(f ACLSetter, p string) error {
	islink, err := isLink(p)
	if err != nil {
		return err
	}

	if islink && ignoreLinks {
		return nil
	}

	// do something to the path
	if err := f(p); err != nil {
		return err
	}

	fi, err := os.Stat(p)
	if err != nil {
		return err
	}
	if fi.IsDir() && recursive {
		dirents, err := ioutil.ReadDir(p)
		if err != nil {
			return err
		}
		for _, child := range dirents {
			p = filepath.Join(p, child.Name())
			if err := apply(f, p); err != nil {
				return err
			}
		}
	}
	return nil
}
