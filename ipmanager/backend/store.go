package backend

import "net"

type Store interface {
	Reserve(network, id string, ip net.IP) (bool, error)
}
