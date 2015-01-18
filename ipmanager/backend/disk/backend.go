package disk

import (
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
)

var defaultDataDir = "/var/lib/ipmanager/networks"

type Store struct {
	dataDir string
}

func New() (*Store, error) {
	if err := os.MkdirAll(defaultDataDir, 0644); err != nil {
		return nil, err
	}
	return &Store{defaultDataDir}, nil
}

func (s *Store) Reserve(network, id string, ip net.IP) (bool, error) {
	dst := filepath.Join(s.dataDir, network)
	if err := os.MkdirAll(dst, 0644); err != nil {
		return false, err
	}
	fname := filepath.Join(dst, ip.String())
	f, err := os.OpenFile(fname, os.O_RDWR|os.O_EXCL|os.O_CREATE, 0644)
	if os.IsExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if _, err := f.WriteString(id); err != nil {
		f.Close()
		os.Remove(f.Name())
		return false, err
	}
	if err := f.Close(); err != nil {
		os.Remove(f.Name())
		return false, err
	}
	return true, nil
}

func (s *Store) Release(network string, ip net.IP) error {
	return os.Remove(filepath.Join(s.dataDir, ip.String()))
}

func (s *Store) ReleaseByContainerID(network string, id string) error {
	err := filepath.Walk(s.dataDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() && path != s.dataDir {
			return nil
		}
		data, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}
		if string(data) == id {
			if err := os.Remove(path); err != nil {
				return err
			}
		}
		return nil
	})
	return err
}
