package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"runtime"
	"strings"
	"syscall"

	"github.com/appc/spec/schema/types"
	"github.com/coreos/rocket/Godeps/_workspace/src/github.com/vishvananda/netlink"

	"github.com/coreos/rocket/networking/ipam"
	"github.com/coreos/rocket/networking/util"
)

func init() {
	// this ensures that main runs only on main thread (thread group leader).
	// since namespace ops (unshare, setns) are done for a single thread, we
	// must ensure that the goroutine does not jump from OS thread to thread
	runtime.LockOSThread()
}

func argsHasDefault(args string) bool {
	argv := strings.Split(args, ",")
	for _, a := range argv {
		if a == "default" {
			return true
		}
	}
	return false
}

func cmdAdd(contID, netns, netConf, ifName, args string) error {
	var hostVethName string

	cid, err := types.NewUUID(contID)
	if err != nil {
		return fmt.Errorf("Error parsing ContainerID: %v", err)
	}

	ipn, gw, err := ipam.AllocIP(*cid, netConf, ifName, args)
	if err != nil {
		return err
	}

	err = util.WithNetNSPath(netns, func(hostNS *os.File) error {
		entropy := contID + ifName

		hostVeth, contVeth, err := util.SetupVeth(entropy, ifName, ipn, hostNS)
		if err != nil {
			return err
		}

		if argsHasDefault(args) {
			if err = util.AddDefaultRoute(gw, contVeth); err != nil {
				return fmt.Errorf("failed to add default route: %v", err)
			}
		}

		hostVethName = hostVeth.Attrs().Name
		return err
	})

	// hostVeth moved namespaces and will have a new ifindex
	hostVeth, err := netlink.LinkByName(hostVethName)
	if err != nil {
		return fmt.Errorf("failed to lookup %q: %v", hostVeth.Attrs().Name, err)
	}


	// On the host we route traffic for the allocated IP to the container
	ipn.Mask = net.CIDRMask(32, 32)

	if err = util.AddRoute(ipn, nil, hostVeth); err != nil {
		return fmt.Errorf("failed to add route on host: %v", err)
	}

	os.Stdout.Write([]byte(ipn.String()))

	return nil
}

func cmdDel(contID, netns, netConf, ifName, args string) error {
	// switch to the container namespace
	contNS, err := os.Open(netns)
	if err != nil {
		return fmt.Errorf("Failed to open %v: %v", netns, err)
	}

	if err = util.SetNS(contNS, syscall.CLONE_NEWNET); err != nil {
		return fmt.Errorf("Error switching to ns %v: %v", netns, err)
	}

	iface, err := netlink.LinkByName(ifName)
	if err != nil {
		return fmt.Errorf("Failed to lookup %q: %v", ifName, err)
	}

	if err = netlink.LinkDel(iface); err != nil {
		return fmt.Errorf("Failed to delete %q: %v", ifName, err)
	}

	return nil
}

func usage() int {
	fmt.Fprintln(os.Stderr, "USAGE: add|del CONTAINER-ID NETNS NET-CONF IF-NAME ARGS")
	return 1
}

func main() {
	if len(os.Args) != 7 {
		os.Exit(usage())
	}

	var err error

	switch os.Args[1] {
	case "add":
		err = cmdAdd(os.Args[2], os.Args[3], os.Args[4], os.Args[5], os.Args[6])

	case "del":
		err = cmdDel(os.Args[2], os.Args[3], os.Args[4], os.Args[5], os.Args[6])

	default:
		os.Exit(usage())
	}

	if err != nil {
		log.Print(err)
		os.Exit(1)
	}
}
