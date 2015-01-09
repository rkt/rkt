package ipam

// ATTN: This is mostly throw away code. It'll be replaced
// by proper ip mgt plugins.

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"math/big"
	"net"
	"strings"

	"github.com/coreos/rocket/network/util"
)

type options struct {
	ipRange *net.IPNet
	ip      net.IP
}

func ipAdd(ip net.IP, val uint) net.IP {
	n := binary.BigEndian.Uint32(ip.To4())
	n += uint32(val)

	nip := make([]byte, 4)
	binary.BigEndian.PutUint32(nip, n)
	return net.IP(nip)
}

func allocIP(ipn *net.IPNet) (*net.IPNet, error) {
	ones, bits := ipn.Mask.Size()
	zeros := bits - ones
	rng := (1 << uint(zeros)) - 2 // (reduce for gw, bcast)

	n, err := rand.Int(rand.Reader, big.NewInt(int64(rng)))
	if err != nil {
		return nil, err
	}

	offset := uint(n.Uint64() + 1)

	return &net.IPNet{
		IP: ipAdd(ipn.IP, offset),
		Mask: ipn.Mask,
	}, nil
}

func deallocIP(ip net.IP) error {
	return nil
}

func splitArg(arg string) (k, v string) {
	parts := strings.SplitN(arg, "=", 2)

	switch len(parts) {
	case 1:
		k = parts[0]
	case 2:
		k, v = parts[0], parts[1]
	}

	return
}

func parseArgs(args string) (*options, error) {
	argv := strings.Split(args, ",")

	var err error
	opts := &options{}

	for _, arg := range argv {
		k, v := splitArg(arg)
		switch k {
		case "iprange":
			opts.ipRange, err = util.ParseCIDR(v)
			if err != nil {
				return nil, fmt.Errorf("failed to parse iprange arg (%q): %v", v, err)
			}

		case "ip":
			opts.ip = net.ParseIP(v)
			if opts.ip == nil {
				return nil, fmt.Errorf("failed to parse ip arg (%q)", v)
			}
		}
	}

	return opts, nil
}

func AllocIP(contID, netConf, ifName, args string) (*net.IPNet, net.IP, error) {
	opts, err := parseArgs(args)
	if err != nil {
		return nil, nil, err
	}

	if opts.ipRange != nil {
		ipn, err := allocIP(opts.ipRange)
		if err != nil {
			return nil, nil, fmt.Errorf("error allocating IP in %v: %v", ipn, err)
		}
		return ipn, nil, nil
	}

	n := util.Net{}
	if err := util.LoadNet(netConf, &n); err != nil {
		return nil, nil, err
	}

	switch n.IPAlloc.Type {
	case "static":
		_, ipn, err := net.ParseCIDR(n.IPAlloc.Subnet)
		if err != nil {
			// TODO: cleanup
			return nil, nil, fmt.Errorf("error parsing %q conf: ipAlloc.Subnet: %v")
		}

		ipn, err = allocIP(ipn)
		if err != nil {
			// TODO: cleanup
			return nil, nil, fmt.Errorf("error allocating IP in %v: %v", ipn, err)
		}

		return ipn, nil, nil

	default:
		return nil, nil, fmt.Errorf("unsupported IP allocation type")
	}
}

func DeallocIP(contID, netConf, ifName string, ipn *net.IPNet) error {
	return nil
}
