## Using NAT with bridge

The bridge plugin can be configured to create a separate network on the host that will be NAT'ed similar to the default network.
The difference to a ptp configured network is that the pods will be able to communicate directly through the bridge and don't have to pass the host as a gateway.

```json
$ cat /etc/rkt/net.d/10-bridge-nat.conf
{
    "name": "bridge-nat",
    "type": "bridge",
    "bridge": "rkt-bridge-nat",
    "ipMasq": true,
    "isGateway": true,
    "ipam": {
        "type": "host-local",
        "subnet": "10.2.0.0/24",
        "routes": [
                { "dst": "0.0.0.0/0" }
        ]
    }
}
```

Inside the pod the interface configuration looks like this:

```
$ sudo rkt run --net=bridge-nat --interactive --debug kinvolk.io/aci/busybox:1.24
(...)
# ip -4 address
1: lo: <LOOPBACK,UP,LOWER_UP> mtu 65536 qdisc noqueue
    inet 127.0.0.1/8 scope host lo
       valid_lft forever preferred_lft forever
3: eth0@if68: <BROADCAST,MULTICAST,UP,LOWER_UP,M-DOWN> mtu 1500 qdisc noqueue
    inet 10.2.0.2/24 scope global eth0
       valid_lft forever preferred_lft forever
5: eth1@if69: <BROADCAST,MULTICAST,UP,LOWER_UP,M-DOWN> mtu 1500 qdisc noqueue
    inet 172.16.28.2/24 scope global eth1
       valid_lft forever preferred_lft forever
# ip -4 route
default via 10.2.0.1 dev eth0
10.2.0.0/24 dev eth0  src 10.2.0.2
172.16.28.0/24 via 172.16.28.1 dev eth1  src 172.16.28.2
172.16.28.1 dev eth1  src 172.16.28.2
```

Note that the _default-restricted_ network has been loaded in addition to the requested network.
