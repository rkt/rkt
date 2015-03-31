package main

import (
	"encoding/json"
	"errors"
	"os"

	"github.com/coreos/rkt/networking/ipam"
	"github.com/coreos/rkt/networking/ipam/static/backend/disk"
	"github.com/coreos/rkt/networking/util"
)

func main() {
	util.PluginMain(cmdAdd, cmdDel)
}

func cmdAdd(args *util.CmdArgs) error {
	ipamConf, err := NewIPAMConfig(args.NetConf)
	if err != nil {
		return err
	}

	store, err := disk.New(ipamConf.Name)
	if err != nil {
		return err
	}
	defer store.Close()

	allocator, err := NewIPAllocator(ipamConf, store)
	if err != nil {
		return err
	}

	var ipConf *ipam.IPConfig

	switch ipamConf.Type {
	case "static":
		ipConf, err = allocator.Get(args.PodID.String())
	case "static-ptp":
		ipConf, err = allocator.GetPtP(args.PodID.String())
	default:
		return errors.New("Unsupported IPAM plugin type")
	}

	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(ipConf, "", "    ")
	if err != nil {
		return err
	}

	_, err = os.Stdout.Write(data)
	return err
}

func cmdDel(args *util.CmdArgs) error {
	ipamConf, err := NewIPAMConfig(args.NetConf)
	if err != nil {
		return err
	}

	store, err := disk.New(ipamConf.Name)
	if err != nil {
		return err
	}
	defer store.Close()

	allocator, err := NewIPAllocator(ipamConf, store)
	if err != nil {
		return err
	}

	return allocator.Release(args.PodID.String())
}
