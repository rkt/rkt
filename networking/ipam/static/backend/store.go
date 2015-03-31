package backend

import "net"

type Store interface {
	Lock() error
	Unlock() error
	Close() error
	Reserve(id string, ip net.IP) (bool, error)
	Release(ip net.IP) error
	ReleaseByPodID(id string) error
}
