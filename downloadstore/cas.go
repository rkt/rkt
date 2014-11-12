package main

import (
	"fmt"

	"github.com/peterbourgon/diskv"
)

const (
	remoteType int64 = iota
	objectType
	downloadType
)

var otmap = [...]string{
	"remote",
	"object",
	"download",
}

type Blob interface {
	Hash() string
	Marshal() []byte
	Unmarshal([]byte)
	Type() int64
}

type DownloadStore struct {
	stores []*diskv.Diskv
}

func NewDownloadStore() *DownloadStore {
	ds := &DownloadStore{}
	ds.stores = make([]*diskv.Diskv, len(otmap))

	for i, p := range otmap {
		ds.stores[i] = diskv.New(diskv.Options{
			BasePath:     "data/"+p,
			Transform:    blockTransform,
			CacheSizeMax: 1024 * 1024, // 1MB
		})
	}

	return ds
}

func (ds DownloadStore) Dump(hex bool) {
	for _, s := range ds.stores {
		var keyCount int
		for key := range s.Keys() {
			val, err := s.Read(key)
			if err != nil {
				panic(fmt.Sprintf("key %s had no value", key))
			}
			if len(val) > 128 {
				val = val[:128]
			}
			out := string(val)
			if hex {
				out = fmt.Sprintf("%x", val)
			}
			fmt.Printf("%s/%s: %s\n", s.BasePath, key, out)
			keyCount++
		}
		fmt.Printf("%d total keys\n", keyCount)
	}
}

func (ds DownloadStore) Store(b Blob) {
	ds.stores[b.Type()].Write(b.Hash(), b.Marshal())
}

func (ds DownloadStore) Get(b Blob) error {
	buf, err := ds.stores[b.Type()].Read(b.Hash())
	if err != nil {
		return err
	}

	b.Unmarshal(buf)

	return nil
}
