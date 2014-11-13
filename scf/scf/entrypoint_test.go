package scf

import (
	"fmt"
	"testing"
)

func TestLoadExecFile(t *testing.T) {
	json :=	[]byte(`{"SCFVersion": "1","Name": "example.com/ourapp-1.0.0","OS": "linux","Arch": "amd64","Exec": ["/usr/bin/mysql"],"Type": "oneshot","User": "100","Group": "300","Environment": {"MYSQL_DEBUG": "true"},"MountPoints": [{"Name": "database", "Path": "/var/lib/mysql", "ReadOnly": false}],"Isolators": {"PrivateNetwork": "true","CpuShares": "20","MemoryLimit": "1G","CapabilityBoundingSet": "CAP_NET_BIND_SERVICE CAP_SYS_ADMIN"}}`)

	ef, err := loadExecFile(json)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("Unit: %s\n", ef.Name);

	/* TODO: verify things are set to what we expect from the supplied json.. */
	for _, x := range ef.Exec {
		fmt.Printf(" exec: %s\n", x);
	}

	for k, v := range ef.Env {
		fmt.Printf(" env: %v=%s\n", k, v);
	}

	for _, v := range ef.Mounts {
		fmt.Printf(" mountpoint: %s @ %s readonly=%v\n", v.Name, v.Path, v.RdOnly);
	}

	for k, v := range ef.Isols {
		fmt.Printf(" isolator: %v=%s\n", k, v);
	}
}
