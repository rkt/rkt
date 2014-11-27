package schema

import (
	"encoding/json"
	"errors"

	"github.com/coreos-inc/rkt/app-container/schema/types"
)

type FileSetManifest struct {
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

func NewFileSetManifest(name string) *FileSetManifest {
	return &FileSetManifest{
		ACVersion: AppContainerVersion,
		ACKind:    "FileSetManifest",
		OS:        "linux",
		Arch:      "amd64",
		Name:      types.ACName(name),
	}
}

type fileSetManifest FileSetManifest

func (fsm *FileSetManifest) assertValid() error {
	if fsm.ACKind != "FileSetManifest" {
		return types.ACKindError(`missing or bad ACKind (must be "FileSetManifest")`)
	}
	if fsm.OS != "linux" {
		return errors.New(`missing or bad OS (must be "linux")`)
	}
	if fsm.Arch != "amd64" {
		return errors.New(`missing or bad Arch (must be "amd64")`)
	}
	return nil
}

func (fsm *FileSetManifest) UnmarshalJSON(data []byte) error {
	f := fileSetManifest{}
	err := json.Unmarshal(data, &f)
	if err != nil {
		return err
	}
	nfsm := FileSetManifest(f)
	if err := nfsm.assertValid(); err != nil {
		return err
	}
	*fsm = nfsm
	return nil
}

func (fsm FileSetManifest) MarshalJSON() ([]byte, error) {
	if err := fsm.assertValid(); err != nil {
		return nil, err
	}
	return json.Marshal(fileSetManifest(fsm))
}
