# rkt image

## rkt image list

You can get a list of images in the local store with their keys, app names and import times.

```
# rkt image list
KEY                                                                     APPNAME                         IMPORTTIME                              LATEST
sha512-fa1cb92dc276b0f9bedf87981e61ecde93cc16432d2441f23aa006a42bb873df coreos.com/etcd:v2.0.0          2015-07-10 10:14:37.323 +0200 CEST      false
sha512-a03f6bad952b30ca1875b1b179ab34a0f556cfbf3893950f59c408992d1bc891 coreos.com/rkt/stage1:0.7.0     2015-07-12 20:27:56.041 +0200 CEST      false
```

## rkt image rm

Given an image key you can remove it from the local store.

```
# rkt image rm sha512-a03f6bad952b30ca1875b1b179ab34a0f556cfbf3893950f59c408992d1bc891
rkt: successfully removed aci for imageID: "sha512-a03f6bad952b30ca1875b1b179ab34a0f556cfbf3893950f59c408992d1bc891"
rkt: 1 image(s) successfully remove
```

## rkt image export

There are cases where you might want to export the ACI from the store to copy to another machine, file server, etc.

```
# rkt image export coreos.com/etcd etcd.aci
$ tar xvf etcd.aci
```

NOTES:
- A matching image must be fetched before doing this operation, rkt will not attempt to download an image first, this subcommand will incur no-network I/O.
- The exported ACI file might be different than the original one because rkt image export always returns uncompressed ACIs.


## rkt image extract/render

For debugging or inspection you may want to extract an ACI to a directory on disk. There are a few different options depending on your use case but the basic command looks like this:

```
# rkt image extract coreos.com/etcd etcd-extracted
# find etcd-extracted
etcd-extracted
etcd-extracted/manifest
etcd-extracted/rootfs
etcd-extracted/rootfs/etcd
etcd-extracted/rootfs/etcdctl
...
```

NOTE: Like with rkt image export, a matching image must be fetched before doing this operation.

Now there are some flags that can be added to this:

To get just the rootfs use:

```
# rkt image extract --rootfs-only coreos.com/etcd etcd-extracted
# find etcd-extracted
etcd-extracted
etcd-extracted/etcd
etcd-extracted/etcdctl
...
```

If you want the image rendered as it would look ready-to-run inside of the rkt stage2 then use `rkt image render`. NOTE: this will not use overlayfs or any other mechanism. This is to simplify the cleanup: to remove the extracted files you can run a simple `rm -Rf`.

## rkt image cat-manifest

For debugging or inspection you may want to extract an ACI manifest to stdout.

```
# rkt image cat-manifest --pretty-print coreos.com/etcd
{
  "acVersion": "0.7.0",
  "acKind": "ImageManifest",
...
```
