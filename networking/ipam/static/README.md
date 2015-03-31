# static IP address manager

static IPAM allocates static IPv4 and IPv6 addresses for pods.

## Usage

### Obtain an IP

Given the following network configuration:

`/etc/rkt/net.d/default.conf`

```
{
    "name": "default",
    "subnet": "203.0.113.0/24"
}
```

#### Using the command line interface

```
$ export RKT_NETPLUGIN_COMMAND=ADD
$ export RKT_NETPLUGIN_PODID=f81d4fae-7dec-11d0-a765-00a0c91e6bf6
$ export RKT_NETPLUGIN_NETCONF=/etc/rkt/net.d/default.conf
$ ./static
```

```
{
    "ip": "203.0.113.1/24"
}
```

## Backends

By default ipmanager stores IP allocations on the local filesystem using the IP address as the file name and the pod ID as contents. For example:

```
$ ls /var/lib/rkt/networks/default
```
```
203.0.113.1	203.0.113.2
```

```
$ cat /var/lib/rkt/networks/default/203.0.113.1
```
```
f81d4fae-7dec-11d0-a765-00a0c91e6bf6
```

## Configuration Files


`/etc/rkt/net.d/ipv6.conf`

```
{
	"name": "ipv6",
	"subnet": "3ffe:ffff:0:01ff::/64",
	"range-start": "3ffe:ffff:0:01ff::0010",
	"range-end": "3ffe:ffff:0:01ff::0020",
	"routes": [
		"3ffe:ffff:0:01ff::1/64"
	]
}
```

`/etc/rkt/net.d/ipv4.conf`

```
{
    "name": "ipv4",
    "subnet": "203.0.113.1/24",
    "range-start": "203.0.113.10",
    "range-end": "203.0.113.20",
    "routes": [
        "203.0.113.0/24"
    ]
}
```
