/* standard container format entry point */
package scf

import (
	"os"
	"fmt"
	"encoding/json"
)

type MountPoint struct {
	Name	string			`json:"Name"`
	Path	string			`json:"Path"`
	RdOnly	bool			`json:"ReadOnly"`
}

type Port struct {
	Name	string			`json:"Name"`
	Proto	string			`json:"Protocol"`
	Port	uint16			`json:"Port"`
}

type ExecFile struct {
	Version	string			`json:"SCFVersion"`
	Name	string			`json:"Name"`
	OS	string			`json:"OS"`
	Arch	string			`json:"Arch"`
	Exec	[]string		`json:"Exec,omitempty"`
	Type	string			`json:"Type,omitempty"`
	User	string			`json:"User,omitempty"`
	Group	string			`json:"Group,omitempty"`
	Env	map[string]string	`json:"Environment,omitempty"`
	Mounts	[]MountPoint		`json:"MountPoints,omitempty"`
	Isols	map[string]string	`json:"Isolators,omitempty"`
}

/* load and validate an SCF entrypoint exec file */
func loadExecFile(blob []byte) (*ExecFile, error) {
	ef := &ExecFile{}

	err := json.Unmarshal(blob, ef)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to unmarshal execfile: %v\n", err)
		return nil, err
	}

	/* ensure it has the minimum usable fields */
	if ef.Version != "1" {
		fmt.Fprintf(os.Stderr, "Unsupported version: %v\n", ef.Version)
		return nil, err
	}

	if ef.Name == "" {
		fmt.Fprintf(os.Stderr, "Name required\n")
		return nil, err
	}

	/* TODO any further validation */
	return ef, nil
}
