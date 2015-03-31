package main

import (
	"fmt"
	"net"

	"github.com/coreos/rkt/networking/ipam"
	"github.com/coreos/rkt/networking/ipam/static/backend"
	"github.com/coreos/rkt/networking/util"
)

type IPAllocator struct {
	start net.IP
	end   net.IP
	last  net.IP
	ipnet *net.IPNet
	conf  *IPAMConfig
	store backend.Store
}

func NewIPAllocator(conf *IPAMConfig, store backend.Store) (*IPAllocator, error) {
	var (
		start net.IP
		end   net.IP
		err   error
	)
	_, ipnet, err := net.ParseCIDR(conf.Subnet)
	if err != nil {
		return nil, err
	}
	start, end = networkRange(ipnet)
	start = util.NextIP(start)
	end = util.PrevIP(end)

	if conf.RangeStart != "" {
		start, err = validateRangeIP(conf.RangeStart, ipnet)
		if err != nil {
			return nil, err
		}
	}
	if conf.RangeEnd != "" {
		end, err = validateRangeIP(conf.RangeEnd, ipnet)
		if err != nil {
			return nil, err
		}
	}

	return &IPAllocator{start, end, util.PrevIP(start), ipnet, conf, store}, nil
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

// Returns newly allocated IP along with its config
func (a *IPAllocator) Get(id string) (*ipam.IPConfig, error) {
	a.store.Lock()
	defer a.store.Unlock()

	var gw net.IP
	if a.conf.Gateway != "" {
		gw = net.ParseIP(a.conf.Gateway)
	}

	routes, err := parseIPNets(a.conf.Routes)
	if err != nil {
		return nil, err
	}

	ip := a.nextIP()
	seen := newIP(ip)
	for {
		// don't allocate gateway IP
		if gw != nil && ip.Equal(gw) {
			continue
		}

		reserved, err := a.store.Reserve(id, ip)
		if err != nil {
			return nil, err
		}
		if reserved {
			break
		}
		ip = a.nextIP()
		if seen.Equal(ip) {
			return nil, fmt.Errorf("no ip addresses available in network: %s", a.conf.Name)
		}
	}

	ipnet := net.IPNet{ip, a.ipnet.Mask}
	alloc := &ipam.IPConfig{
		IP:      &ipnet,
		Gateway: gw,
		Routes:  routes,
	}
	return alloc, nil
}

// Allocates both an IP and the Gateway IP, i.e. a /31
// This is used for Point-to-Point links
func (a *IPAllocator) GetPtP(id string) (*ipam.IPConfig, error) {
	a.store.Lock()
	defer a.store.Unlock()

	routes, err := parseIPNets(a.conf.Routes)
	if err != nil {
		return nil, err
	}

	gw := a.nextIP()
	ip := net.IP{}

	seen := newIP(gw)

	for {
		if evenIP(gw) {
			// we're looking for unreserved even, odd pair
			reserved, err := a.store.Reserve(id, gw)
			if err != nil {
				return nil, err
			}
			if reserved {
				ip = a.nextIP()
				reserved, err := a.store.Reserve(id, ip)
				if err != nil {
					return nil, err
				}
				if reserved {
					// found them both!
					break
				}
			}
		}

		gw = a.nextIP()
		if seen.Equal(gw) {
			return nil, fmt.Errorf("no ip addresses available in network: %s", a.conf.Name)
		}
	}

	_, bits := a.ipnet.Mask.Size()
	mask := net.CIDRMask(bits-1, bits)

	alloc := &ipam.IPConfig{
		IP:      &net.IPNet{ip, mask},
		Gateway: gw,
		Routes:  routes,
	}
	return alloc, nil
}

// Releases all IPs allocated for the pod with given ID
func (a *IPAllocator) Release(id string) error {
	a.store.Lock()
	defer a.store.Unlock()

	return a.store.ReleaseByPodID(id)
}

func (a *IPAllocator) nextIP() net.IP {
	if a.last.Equal(a.end) {
		a.last = a.start
		return a.start
	}
	a.last = util.NextIP(a.last)
	return a.last
}

func newIP(ip net.IP) net.IP {
	n := make(net.IP, len(ip))
	copy(n, ip)
	return n
}

func networkRange(ipnet *net.IPNet) (net.IP, net.IP) {
	var end net.IP
	for i := 0; i < len(ipnet.IP); i++ {
		end = append(end, ipnet.IP[i]|^ipnet.Mask[i])
	}
	return ipnet.IP, end
}

func evenIP(ip net.IP) bool {
	i := ip.To4()
	if i == nil {
		i = ip.To16()
		if i == nil {
			panic("IP is not v4 or v6")
		}
	}

	return i[len(i)-1]%2 == 0
}

func parseIPNets(ipnets []string) ([]net.IPNet, error) {
	n := []net.IPNet{}
	for _, s := range ipnets {
		ipn, err := util.ParseCIDR(s)
		if err != nil {
			return nil, err
		}
		n = append(n, *ipn)
	}
	return n, nil
}
