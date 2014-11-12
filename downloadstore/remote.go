package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io"
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
	return sha1sum(r.Name)
}

func (r Remote) Type() int64 {
	return remoteType
}

// TODO: add locking
func (r Remote) Download(ds DownloadStore) (*Remote, error) {
	var b bytes.Buffer
	hash := sha1.New()

	res, err := http.Get(r.Name)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	// TODO: use go routines to parallelize this pipeline
	_, err = io.Copy(&b, io.TeeReader(res.Body, hash))
	if err != nil {
		return nil, err
	}
	err = ds.stores[downloadType].WriteStream(r.Hash(), &b, true)
	if err != nil {
		return nil, err
	}

	rs, err := ds.stores[downloadType].ReadStream(r.Hash(), false)
	if err != nil {
		return nil, err
	}
	key := fmt.Sprintf("%x", hash.Sum(nil))
	err = ds.stores[objectType].WriteStream(key, rs, true)
	if err != nil {
		return nil, err
	}

	ds.stores[downloadType].Erase(r.Hash())
	r.File = key
	ds.stores[remoteType].Write(r.Hash(), r.Marshal())

	return &r, nil
}
