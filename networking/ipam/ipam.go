// Copyright 2015 CoreOS, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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

	"github.com/appc/spec/schema/types"

	"github.com/coreos/rocket/networking/util"
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

func allocIP(ipn *net.IPNet) (net.IP, error) {
	ones, bits := ipn.Mask.Size()
	zeros := bits - ones
	rng := (1 << uint(zeros)) - 2 // (reduce for gw, bcast)

	n, err := rand.Int(rand.Reader, big.NewInt(int64(rng)))
	if err != nil {
		return nil, err
	}

	offset := uint(n.Uint64() + 1)
	return ipAdd(ipn.IP, offset), nil
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

// AllocIP allocates an IP in a given range.
func AllocIP(contID types.UUID, netConf, ifName, args string) (*net.IPNet, net.IP, error) {
	opts, err := parseArgs(args)
	if err != nil {
		return nil, nil, err
	}

	if opts.ipRange != nil {
		ip, err := allocIP(opts.ipRange)
		if err != nil {
			return nil, nil, fmt.Errorf("error allocating IP in %v: %v", opts.ipRange, err)
		}

		return &net.IPNet{
			IP:   ip,
			Mask: opts.ipRange.Mask,
		}, nil, nil
	}

	n := util.Net{}
	if err := util.LoadNet(netConf, &n); err != nil {
		return nil, nil, err
	}

	switch n.IPAlloc.Type {
	case "static":
		_, rng, err := net.ParseCIDR(n.IPAlloc.Subnet)
		if err != nil {
			// TODO: cleanup
			return nil, nil, fmt.Errorf("error parsing %q conf: ipAlloc.Subnet: %v", netConf, err)
		}

		ip, err := allocIP(rng)
		if err != nil {
			// TODO: cleanup
			return nil, nil, fmt.Errorf("error allocating IP in %v: %v", rng, err)
		}

		return &net.IPNet{
			IP:   ip,
			Mask: rng.Mask,
		}, nil, nil

	default:
		return nil, nil, fmt.Errorf("unsupported IP allocation type")
	}
}

// AllocPtP allocates a /31 for point-to-point links.
func AllocPtP(contID types.UUID, netConf, ifName, args string) ([2]net.IP, error) {
	ipn, _, err := AllocIP(contID, netConf, ifName, args)
	if err != nil {
		return [2]net.IP{nil, nil}, err
	}

	mask := net.CIDRMask(31, 32)
	first := ipn.IP.Mask(mask)
	second := ipAdd(first, 1)

	return [2]net.IP{first, second}, nil
}

// DeallocIP is a no-op.
func DeallocIP(contID types.UUID, netConf, ifName string, ipn *net.IPNet) error {
	return nil
}
