# Metadata service example application

This directory includes an example application interacting with the rkt [metadata service](../../subcommands/metadata-service.md). It can sign and verify files against the metadata service following the [Identity Endpoint](https://github.com/appc/spec/blob/v0.8.11/spec/ace.md#identity-endpoint) of the [appc spec](https://github.com/appc/spec).

The signature file format is just the base64-encoded signature returned by the metadata service.

Note that this is just an example application: it doesn't work for big files and is not suitable for production use.

## Compilation

```
$ CGO_ENABLED=0 go build -o mds-example
```

## Trying it out

We'll start two pods and send a file from one to the other.
We'll use our example application to sign the file against the metadata service, then we'll send the file and the signature to the receiver pod and verify its integrity and authenticity using our example application too.

Assuming we're in this directory and we've [compiled](#compilation) the example application and the metadata service is running, let's start two busybox pods with our app mounted inside: "POD a" and "POD b"

```shell
$ sudo rkt run --mds-register --interactive --volume=mds-example,kind=host,source=$PWD/mds-example kinvolk.io/aci/busybox:1.24 --mount volume=mds-example,target=/bin/mds-example
[POD a] / #
```

```shell
$ sudo rkt run --mds-register --interactive --volume=mds-example,kind=host,source=$PWD/mds-example kinvolk.io/aci/busybox:1.24 --mount volume=mds-example,target=/bin/mds-example
[POD b] / #
```

In "POD a", we'll create a message and sign it using our example application:

```
[POD a] / # echo "Very trustworthy message" > msg.txt
[POD a] / # mds-example sign --file=msg.txt --signature=msg.sig
[POD a] / # ls -l msg*
-rw-------    1 root     root           125 Dec 18 17:34 msg.sig
-rw-r--r--    1 root     root            25 Dec 18 17:34 msg.txt
```

Now we'll transfer the message to "POD b":

```
# let's find out the IP address of POD b
[POD b] / # ip a
(...)
    inet 172.16.28.84/24 scope global eth0
(...)
[POD b] / # nc -l -p 9090 > msg.txt

# switch to POD a to send the message
[POD a] / # nc 172.16.28.84 9090 < msg.txt

# switch to POD b
[POD b] / # nc -l -p 9090 > msg.sig

# switch to POD a to send the signature
[POD a] / # nc 172.16.28.84 9090 < msg.sig

# find out the UUID of POD a
[POD a] / # echo $(wget -q -O - $AC_METADATA_URL/acMetadata/v1/pod/uuid)
d1703a6a-568a-48ee-b84b-4ccd803743dd
```

And finally, we'll verify the message in "POD b":

```
[POD b] / # cat msg.txt
Very trustworthy message
[POD b] / # mds-example verify --file msg.txt --uuid=d1703a6a-568a-48ee-b84b-4ccd803743dd --signature=msg.sig
signature OK
```
