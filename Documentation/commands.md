# rkt Commands

Work in progress. Please contribute if you see an area that needs more detail.

## Downloading Images (ACIs)

[aci-images]: https://github.com/appc/spec/blob/master/spec/aci.md#app-container-image
[appc-discovery]: https://github.com/appc/spec/blob/master/spec/discovery.md#app-container-image-discovery

rkt runs applications packaged as [Application Container Images (ACI)][aci-images] an open-source specification. ACIs consist of the root filesystem of the application container, a manifest and an optional signature.

ACIs are named with a URL-like structure. This naming scheme allows for a decentralized discovery of ACIs, related signatures and public keys. rkt uses these hints to execute [meta discovery][appc-discovery].

* [trust](subcommands/trust.md)
* [fetch](subcommands/fetch.md)

## Running Pods

rkt can run ACIs based on name, hash, local file on disk or URL. If an ACI hasn't been cached on disk, rkt will attempt to find and download it.
If you want to use the [metadata service](https://github.com/appc/spec/blob/master/spec/ace.md#app-container-metadata-service), make sure [it is running](https://github.com/coreos/rkt/blob/mastersubcommands/metadata-service.md) and you enable registration with the `--mds-register` flag.

* [run](subcommands/run.md)
* [enter](subcommands/enter.md)
* [prepare](subcommands/prepare.md)
* [run-prepared](subcommands/run-prepared.md)

## Pod inspection and management

rkt provides subcommands to list, get status, and clean its pods.

* [list](subcommands/list.md)
* [status](subcommands/status.md)
* [gc](subcommands/gc.md)
* [rm](subcommands/rm.md)
* [cat-manifest](subcommands/cat-manifest.md)

## Interacting with the local image store

rkt provides subcommands to list, inspect and export images in its local store.

* [image](subcommands/image.md)

## Metadata Service

The metadata service helps running apps introspect their execution environment and assert their pod identity.

* [metadata-service](subcommands/metadata-service.md)

## Logging

By default, rkt will send logs directly to stdout/stderr, allowing them to be captured by the invoking process.
On host systems running systemd, rkt will attempt to integrate with journald on the host.
In this case, the logs can be accessed directly via journalctl.

#### Accessing logs via journalctl

To get the logs of a running pod, you need to get the pod's machine name. You can use machinectl:

```
$ machinectl
MACHINE                                  CLASS     SERVICE
rkt-f241c969-1710-445a-8129-d3a7ffdd9a60 container nspawn

1 machines listed.
```

or `rkt list --full`

```
# rkt list --full
UUID					                APP	    ACI 	STATE	NETWORKS
f241c969-1710-445a-8129-d3a7ffdd9a60	busybox	busybox	running
```

Pod's machine name will be the pod's UUID with a `rkt-` prefix.

Then you can use systemd's journalctl:

```
# journalctl -M rkt-f241c969-1710-445a-8129-d3a7ffdd9a60

[...]
```
