package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/coreos/rocket/networking/ipam"
	"github.com/coreos/rocket/networking/ipam/static/backend/disk"
)

func main() {
	cmd, containerID, netConf := lookupParams()

	ipamConf, err := NewIPAMConfig(netConf)
	if err != nil {
		log.Fatal(err)
	}

	store, err := disk.New(ipamConf.Name)
	if err != nil {
		log.Fatal(err)
	}
	defer store.Close()

	allocator, err := NewIPAllocator(ipamConf, store)
	if err != nil {
		log.Fatal(err)
	}

	switch cmd {
	case "ADD":
		cmdAdd(allocator, containerID, ipamConf.Type)

	case "DEL":
		allocator.Release(containerID)
	}
}

func lookupParams() (cmd, containerID, netConf string) {
	names := []string{"RKT_NETPLUGIN_COMMAND", "RKT_NETPLUGIN_CONTID", "RKT_NETPLUGIN_NETCONF"}
	values := make([]string, len(names))

	for i, n := range names {
		values[i] = os.Getenv(n)
		if values[i] == "" {
			log.Fatalf("%q environment variable is misssing", n)
		}
	}

	cmd, containerID, netConf = values[0], values[1], values[2]
	return
}

func cmdAdd(allocator *IPAllocator, containerID, plugin string) {
	var ipConf *ipam.IPConfig
	var err error

	switch plugin {
	case "static":
		ipConf, err = allocator.Get(containerID)
	case "static-ptp":
		ipConf, err = allocator.GetPtP(containerID)
	default:
		log.Fatal("Unsupported IPAM plugin type")
	}

	if err != nil {
		log.Fatal(err)
	}

	data, err := json.MarshalIndent(ipConf, "", "    ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(data))
}
