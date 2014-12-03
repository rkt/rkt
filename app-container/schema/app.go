package schema

import (
	"encoding/json"
	"errors"

	"github.com/coreos/rocket/app-container/schema/types"
)

const (
	ACIExtension = ".aci"
)

type AppManifest struct {
	ACVersion     types.SemVer         `json:"acVersion"`
	ACKind        types.ACKind         `json:"acKind"`
	Name          types.ACName         `json:"name"`
	Version       types.ACName         `json:"version"`
	OS            types.ACName         `json:"os"`
	Arch          types.ACName         `json:"arch"`
	Exec          []string             `json:"exec"`
	EventHandlers []types.EventHandler `json:"eventHandlers"`
	User          string               `json:"user"`
	Group         string               `json:"group"`
	Environment   map[string]string    `json:"environment"`
	MountPoints   []types.MountPoint   `json:"mountPoints"`
	Ports         []types.Port         `json:"ports"`
	Isolators     []types.Isolator     `json:"isolators"`
	Annotations   types.Annotations    `json:"annotations"`
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
	if am.ACVersion.Empty() {
		return errors.New(`acVersion must be set`)
	}
	if am.Name.Empty() {
		return errors.New(`name must be set`)
	}
	if am.Version.Empty() {
		return errors.New(`version must be set`)
	}
	if am.OS.String() != "linux" {
		return errors.New(`missing or bad OS (must be "linux")`)
	}
	if am.Arch.String() != "amd64" {
		return errors.New(`missing or bad Arch (must be "amd64")`)
	}
	if len(am.Exec) < 1 {
		return errors.New(`Exec cannot be empty`)
	}
	// TODO(jonboulle): assert hashes is not empty?
	return nil
}
