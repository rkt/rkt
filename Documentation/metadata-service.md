# Metadata Service

## Overview

The metadata service is designed to help running apps introspect their execution environment and assert their pod identity.
In particular, the metadata service exposes the contents of the pod and image manifests and well as a convenient method of looking up annotations.
Finally, the metadata service provides a pod with cryptographically verifiable identity.

## Running metadata service

The metadata service is implemented as `rkt metadata-service` command.
When started, it will listen for registration events over Unix socket on /run/rkt/metadata-svc.sock.
For systemd based distributions, it also supports the [systemd socket activation](http://0pointer.de/blog/projects/socket-activation.html).
If using socket activation, keep the socket named /run/rkt/metadata-svc.sock as `rkt run` uses this name during registration.
Please note that when started under socket activation, the metadata service will not remove the socket on exit.
Use `RemoveOnStop` directive in .socket file to cleanup.

In addition to listening on a Unix socket, the metadata service will also listen on a TCP port.
When contacting the metadata service, the apps utilize this port.
In order to avoid exposing host IP to the pods, the metadata service installs an iptables rule to redirect traffic destined for 169.254.169.255:80 to itself.

## Launching metadata service with `rkt run`

For non-production use, it is often convenient to launch the metadata service inline with running a pod.
`rkt run` accepts a `--spawn-metadata-svc` command line argument to launch the metadata service prior to running a pod.
When started in this fashion, the metadata service will exit when there are no more pods registered with it.
Please note that it is strongly recommended to only use this mode when experimenting with rkt on command line.

## Using the metadata service
See [App Container specification](https://github.com/appc/spec/blob/master/SPEC.md#app-container-metadata-service) for more information about the metadata service including a list of supported endpoints and their usage.
