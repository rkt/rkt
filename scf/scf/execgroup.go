/* execution group manifest */
package scf

import (
	"os"
	"fmt"
	"path/filepath"
	"io/ioutil"
	"encoding/json"

	"strings"
)

const (
	ExecGroupPath = "/exec-group"
	Stage2Path =	"/stage1/opt/stage2"
)


type ExecGroup struct {
	Manifest	*Manifest
	Units		[]*ExecFile
}


type ExecUnit struct {
	ID	string				`json:"ID"`
	Prereqs	[]string			`json:"Before,omitempty"`
}

type Volume struct {
	Path	string				`json:"Path,omitempty"`
}

type Manifest struct {
	Version	string				`json:"SEEVersion"`
	UID	string				`json:"UID"`
	Units	map[string]ExecUnit		`json:"Group,omitempty"`
	Vols	map[string]Volume		`json:"Volumes,omitempty"`
}

/* load and validate a pcf rcf execution file */
func loadManifest(blob []byte) (*Manifest, error) {
	egm := &Manifest{}

	err := json.Unmarshal(blob, egm)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to unmarshal group manifest: %v\n", err)
		return nil, err
	}

	/* some validation */
	if egm.Version != "1" {
		fmt.Fprintf(os.Stderr, "Unsupported version: %v\n", egm.Version)
		return nil, err
	}

	if egm.UID == "" {
		fmt.Fprintf(os.Stderr, "UID is required\n")
		return nil, err
	}

	/* ensure all ExecUnit.Prereqs refer to valid Units[keys] */
	for _, unit := range egm.Units {
		if unit.ID == "" {
			fmt.Fprintf(os.Stderr, "ID is required\n")
			return nil, err
		}

		if unit.Prereqs != nil {
			for _, req := range unit.Prereqs {
				if _, ok := egm.Units[req]; !ok {
					fmt.Fprintf(os.Stderr, "Invalid prerequisite: %s\n", req)
					return nil, err
				}
			}
		}
	}

	/* TODO any further necessary validation, like detecting circular Befores? */
	return egm, nil
}


/* yanked this from go-systemd, but since the reference stage1 implementation
 * will be spitting out systemd units anyways it will probably go away */
const (
	alpha = `abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ`
	num = `0123456789`
	alphanum = alpha+num
)
// needsEscape checks whether a byte in a potential dbus ObjectPath needs to be escaped
func needsEscape(i int, b byte) bool {
	// Escape everything that is not a-z-A-Z-0-9
	// Also escape 0-9 if it's the first character
	return strings.IndexByte(alphanum, b) == -1 ||
		(i == 0 && strings.IndexByte(num, b) != -1)
}

// PathBusEscape sanitizes a constituent string of a dbus ObjectPath using the
// rules that systemd uses for serializing special characters.
func nameEscape(path string) string {
	// Special case the empty string
	if len(path) == 0 {
		return "_"
	}
	n := []byte{}
	for i := 0; i < len(path); i++ {
		c := path[i]
		if needsEscape(i, c) {
			e := fmt.Sprintf("_%x", c)
			n = append(n, []byte(e)...)
		} else {
			n = append(n, c)
		}
	}
	return string(n)
}


/* load an exec group from path+ExecGroup and all its runnable container units beneath path+/stage1/opt/stage2/$escaped_name/entrypoints/$name */
func LoadExecGroup(path string) (*ExecGroup, error) {
	eg := &ExecGroup{}

	buf, err := ioutil.ReadFile(filepath.Join(path, ExecGroupPath))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed reading execfile: %v\n", err)
		return nil, err
	}

	eg.Manifest, err = loadManifest(buf)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed loading manifest: %v\n", err)
		return nil, err
	}

	for name, _ := range eg.Manifest.Units {
		esc := nameEscape(name)

		/* XXX: here we're trusting name, should probably sanitize it */
		buf, err := ioutil.ReadFile(filepath.Join(path, Stage2Path, esc, "entrypoints", name))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed reading runnable unit \"%s\": %v\n", name, err)
			return nil, err
		}

		rcu, err := loadExecFile(buf)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed loading entry point %s: %v\n", name, err)
			return nil, err
		}
		eg.Units = append(eg.Units, rcu)
	}

	return eg, nil
}
