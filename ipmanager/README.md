# ipmanager

ipmanager allocates static IPv4 and IPv6 addresses for containers.

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
$ ipmanager -c f81d4fae-7dec-11d0-a765-00a0c91e6bf6
```

```
{
    "ip": "203.0.113.1/24"
}
```

```
$ ipmanager -c 1e8df09a-756f-4f3d-9f0f-3000e46598e7
```

```
{
    "ip": "203.0.113.2/24"
}
```

#### Using the HTTP interface

Start ipmanager in daemon mode.

```
$ ipmanager -s
```
```
2015/01/18 18:02:35 starting ipmanager...
```

Request a new IP address using cURL:

```
curl http://127.0.0.1:8080/default -d '{"containerID": "f81d4fae-7dec-11d0-a765-00a0c91e6bf6"}'
```

```
{
    "ip": "203.0.113.1/24"
}
```

## Backends

By default ipmanager stores IP allocations on the local filesystem using the IP address as the file name and the container ID as contents. For example:

```
$ ls /var/lib/ipmanager/networks/default
```
```
203.0.113.1	203.0.113.2
```

```
$ cat /var/lib/ipmanager/networks/default/203.0.113.1
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
	"routers": [
		"3ffe:ffff:0:01ff::1/64"
	],
	"domain-name": "example.com",
	"domain-name-servers": [
		"2001:4860:4860::8888",
        "2001:4860:4860::8844"
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
    "routers": [
        "203.0.113.1"
    ],
    "domain-name": "example.com",
    "domain-name-servers": [
        "8.8.8.8",
        "8.8.4.4"
    ]
}
```
