package schema

import (
	"encoding/json"
	"errors"

	"github.com/coreos-inc/rkt/app-container/schema/types"
)

type AppManifest struct {
	ACVersion     types.SemVer         `json:"acVersion"`
	ACKind        types.ACKind         `json:"acKind"`
	Name          types.ACName         `json:"name"`
	OS            string               `json:"os"`
	Arch          string               `json:"arch"`
	Exec          []string             `json:"exec"`
	EventHandlers []types.EventHandler `json:"eventHandlers"`
	User          string               `json:"user"`
	Group         string               `json:"group"`
	Environment   map[string]string    `json:"environment"`
	MountPoints   []types.MountPoint   `json:"mountPoints"`
	Ports         []types.Port         `json:"ports"`
	Isolators     []types.Isolator     `json:"isolators"`
	// TODO(jonboulle): whattodo about files?
	Files       map[string]types.File `json:"files"`
	Annotations types.Annotations     `json:"annotations"`
}

// appManifest is a model to facilitate extra validation during the
// unmarshalling of the AppManifest
type appManifest AppManifest

func (am *AppManifest) UnmarshalJSON(data []byte) error {
	a := appManifest{}
	err := json.Unmarshal(data, &a)
	if err != nil {
		return err
	}
	nam := AppManifest(a)
	if err := nam.assertValid(); err != nil {
		return err
	}
	if nam.Environment == nil {
		nam.Environment = make(map[string]string)
	}
	*am = nam
	return nil
}

func (am AppManifest) MarshalJSON() ([]byte, error) {
	if err := am.assertValid(); err != nil {
		return nil, err
	}
	return json.Marshal(appManifest(am))
}

// assertValid performs extra assertions on an AppManifest to ensure that
// fields are set appropriately, etc. It is used exclusively when marshalling
// and unmarshalling an AppManifest. Most field-specific validation is
// performed through the individual types being marshalled; assertValid()
// should only deal with higher-level validation.
func (am *AppManifest) assertValid() error {
	if am.ACKind != "AppManifest" {
		return types.ACKindError(`missing or bad ACKind (must be "AppManifest")`)
	}
	if am.OS != "linux" {
		return errors.New(`missing or bad OS (must be "linux")`)
	}
	if am.Arch != "amd64" {
		return errors.New(`missing or bad Arch (must be "amd64")`)
	}
	// TODO(jonboulle): assert hashes is not empty?
	return nil
}
