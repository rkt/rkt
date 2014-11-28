package cas

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func NewRemote(name string, mirrors []string) *Remote {
	// TODO: don't assume the name is a mirror?
	r := &Remote{}
	r.Name = name
	r.Mirrors = []string{name}
	return r
}

type Remote struct {
	Name    string
	Mirrors []string
	ETag    string
	Blob    string
}

func (r Remote) Marshal() []byte {
	m, _ := json.Marshal(r)
	return m
}

func (r *Remote) Unmarshal(data []byte) {
	err := json.Unmarshal(data, r)
	if err != nil {
		panic(err)
	}
}

func (r Remote) Hash() string {
	return sha256sum(r.Name)
}

func (r Remote) Type() int64 {
	return remoteType
}

// TODO: add locking
func (r Remote) Download(ds Store) (*Remote, error) {
	res, err := http.Get(r.Name)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	// TODO(jonboulle): handle http more robustly (redirects?)
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad HTTP status code: %d", res.StatusCode)
	}

	key, err := ds.WriteACI(r.Hash(), res.Body)
	if err != nil {
		return nil, err
	}

	r.Blob = key
	ds.WriteIndex(&r)

	return &r, nil
}
