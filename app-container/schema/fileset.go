package schema

import (
	"encoding/json"
	"errors"

	"github.com/coreos-inc/rkt/app-container/schema/types"
)

type FilesetManifest struct {
	ACVersion    types.SemVer `json:"acVersion"`
	ACKind       types.ACKind `json:"acKind"`
	Name         types.ACName `json:"name"`
	OS           string       `json:"os"`
	Arch         string       `json:"arch"`
	Dependencies []Dependency `json:"dependencies"`
	Files        []string     `json:"files"`
}

type Dependency struct {
	Name types.ACName `json:"name"`
	Hash types.Hash   `json:"hash"`
	Root string       `json:"root"`
}

func NewFilesetManifest(name string) (*FilesetManifest, error) {
	n, err := types.NewACName(name)
	if err != nil {
		return nil, err
	}
	fsm := FilesetManifest{
		ACVersion: AppContainerVersion,
		ACKind:    "FilesetManifest",
		OS:        "linux",
		Arch:      "amd64",
		Name:      *n,
	}
	return &fsm, nil
}

type fileSetManifest FilesetManifest

func (fsm *FilesetManifest) assertValid() error {
	if fsm.ACKind != "FilesetManifest" {
		return types.ACKindError(`missing or bad ACKind (must be "FilesetManifest")`)
	}
	if fsm.OS != "linux" {
		return errors.New(`missing or bad OS (must be "linux")`)
	}
	if fsm.Arch != "amd64" {
		return errors.New(`missing or bad Arch (must be "amd64")`)
	}
	return nil
}

func (fsm *FilesetManifest) UnmarshalJSON(data []byte) error {
	f := fileSetManifest{}
	err := json.Unmarshal(data, &f)
	if err != nil {
		return err
	}
	nfsm := FilesetManifest(f)
	if err := nfsm.assertValid(); err != nil {
		return err
	}
	*fsm = nfsm
	return nil
}

func (fsm FilesetManifest) MarshalJSON() ([]byte, error) {
	if err := fsm.assertValid(); err != nil {
		return nil, err
	}
	return json.Marshal(fileSetManifest(fsm))
}
