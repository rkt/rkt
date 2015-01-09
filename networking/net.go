package network

import (
	"io/ioutil"
	"log"
	"os"
	"path"

	"github.com/coreos/rocket/networking/util"
)

type Net struct {
	util.Net
	args string
}

const RktNetPath = "/etc/rkt-net.conf.d"
const DefaultIPNet = "172.16.28.0/24"

var defaultNet = Net{
	Net: util.Net{
		Name: "default",
		Type: "veth",
	},
	args: "default,iprange=" + DefaultIPNet,
}

func LoadNets() ([]Net, error) {
	dirents, err := ioutil.ReadDir(RktNetPath)
	switch {
	case err == nil:
	case os.IsNotExist(err):
		return nil, nil
	default:
		return nil, err
	}

	var nets []Net

	for _, dent := range dirents {
		if dent.IsDir() {
			continue
		}

		nf := path.Join(RktNetPath, dent.Name())
		n := Net{}
		if err := util.LoadNet(nf, &n); err != nil {
			log.Printf("Error loading %v: %v", nf, err)
			continue
		}

		nets = append(nets, n)
	}

	nets = append(nets, defaultNet)

	return nets, nil }
