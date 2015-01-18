package main

import (
	"fmt"
	"math/big"
	"net"

	"github.com/coreos/rocket/ipmanager/backend"
)

type Allocation struct {
	DomainName        string   `json:"domain-name,omitempty"`
	DomainNameServers []string `json:"domain-name-servers,omitempty"`
	IP                string   `json:"ip"`
	Routers           []string `json:"routers,omitempty"`
}

type IPAllocator struct {
	start   net.IP
	end     net.IP
	last    net.IP
	ipnet   *net.IPNet
	netconf *NetworkConfig
	store   backend.Store
}

func NewIPAllocator(netconf *NetworkConfig, store backend.Store) (*IPAllocator, error) {
	var (
		start net.IP
		end   net.IP
		err   error
	)
	_, ipnet, err := net.ParseCIDR(netconf.Subnet)
	if err != nil {
		return nil, err
	}
	start, end = networkRange(ipnet)
	start = nextIP(start)
	end = prevIP(end)

	if netconf.RangeStart != "" {
		start, err = validateRangeIP(netconf.RangeStart, ipnet)
		if err != nil {
			return nil, err
		}
	}
	if netconf.RangeEnd != "" {
		end, err = validateRangeIP(netconf.RangeEnd, ipnet)
		if err != nil {
			return nil, err
		}
	}

	return &IPAllocator{start, end, prevIP(start), ipnet, netconf, store}, nil
}

func validateRangeIP(s string, ipnet *net.IPNet) (net.IP, error) {
	ip := net.ParseIP(s)
	if ip == nil {
		return nil, fmt.Errorf("invalid ip address: %s", s)
	}
	if !ipnet.Contains(ip) {
		return nil, fmt.Errorf("%s not in network: %s", s, ipnet)
	}
	return ip, nil
}

func (a *IPAllocator) Get(id string) (*Allocation, error) {
	ip := a.nextIP()
	seen := newIP(ip)
	for {
		reserved, err := a.store.Reserve(a.netconf.Name, id, ip)
		if err != nil {
			return nil, err
		}
		if reserved {
			break
		}
		ip = a.nextIP()
		if seen.Equal(ip) {
			return nil, fmt.Errorf("no ip addresses available in network: %s", a.netconf.Name)
		}
	}

	ipnet := net.IPNet{ip, a.ipnet.Mask}
	alloc := &Allocation{
		IP:                ipnet.String(),
		Routers:           a.netconf.Routers,
		DomainNameServers: a.netconf.DomainNameServers,
		DomainName:        a.netconf.DomainName,
	}
	return alloc, nil
}

func (a *IPAllocator) nextIP() net.IP {
	if a.last.Equal(a.end) {
		a.last = a.start
		return a.start
	}
	a.last = nextIP(a.last)
	return a.last
}

func ipToInt(ip net.IP) *big.Int {
	if v := ip.To4(); v != nil {
		return big.NewInt(0).SetBytes(v)
	}
	return big.NewInt(0).SetBytes(ip.To16())
}

func intToIP(i *big.Int) net.IP {
	return net.IP(i.Bytes())
}

func newIP(ip net.IP) net.IP {
	n := make(net.IP, len(ip))
	copy(n, ip)
	return n
}

func nextIP(ip net.IP) net.IP {
	i := ipToInt(ip)
	return intToIP(i.Add(i, big.NewInt(1)))
}

func prevIP(ip net.IP) net.IP {
	i := ipToInt(ip)
	return intToIP(i.Sub(i, big.NewInt(1)))
}

func networkRange(ipnet *net.IPNet) (net.IP, net.IP) {
	var end net.IP
	for i := 0; i < len(ipnet.IP); i++ {
		end = append(end, ipnet.IP[i]|^ipnet.Mask[i])
	}
	return ipnet.IP, end
}
