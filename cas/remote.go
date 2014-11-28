package cas

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/coreos-inc/rkt/app-container/aci"
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
	File    string
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

func (r Remote) Import(ds Store, orig io.Reader) (*Remote, error) {
	var b bytes.Buffer

	// TODO(philips): use go routines to parallelize this pipeline and make
	// the file type detection happen without a second stream
	_, err := io.Copy(&b, orig)
	if err != nil {
		return nil, err
	}
	err = ds.stores[downloadType].WriteStream(r.Hash(), &b, true)
	if err != nil {
		return nil, err
	}

	// Detect the filetype
	rs, err := ds.stores[downloadType].ReadStream(r.Hash(), false)
	if err != nil {
		return nil, err
	}
	defer rs.Close()
	typ, err := aci.DetectFileType(rs)
	if err != nil {
		return nil, err
	}
	rs, err = ds.stores[downloadType].ReadStream(r.Hash(), false)
	if err != nil {
		return nil, err
	}
	defer rs.Close()

	// Generate the hash of the decompressed tar
	dr, err := decompress(rs, typ)
	if err != nil {
		return nil, err
	}
	hash := sha256.New()
	_, err = io.Copy(hash, dr)
	if err != nil {
		return nil, err
	}

	// Store the decompressed tar
	rs, err = ds.stores[downloadType].ReadStream(r.Hash(), false)
	if err != nil {
		return nil, err
	}
	defer rs.Close()
	dr, err = decompress(rs, typ)
	if err != nil {
		return nil, err
	}

	key := fmt.Sprintf("sha256-%x", hash.Sum(nil))
	err = ds.stores[objectType].WriteStream(key, dr, true)
	if err != nil {
		return nil, err
	}

	ds.stores[downloadType].Erase(r.Hash())
	r.File = key
	ds.stores[remoteType].Write(r.Hash(), r.Marshal())

	return &r, nil
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

	return r.Import(ds, res.Body)
}
