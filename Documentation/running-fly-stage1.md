# Running rkt with the *fly* stage1

The *fly* stage1 is an alternative stage1 that runs a single-application ACI with only `chroot`-isolation.


## Motivation

The motivation of the fly feature is to add the ability to run applications with full privileges on the host but still benefit from the image management and discovery from rkt.
The Kubernetes kubelet is one candidate for rkt fly.


## How does it work?

In comparison to the default stage1, there is no process manager involved in the stage1.

The rkt application sets up bind mounts for `/dev`, `/proc`, `/sys`, and the user-provided volumes.
In addition to the bind mounts, An additional *tmpfs* mount is done at `/tmp`.
After the mounts are set up, rkt `chroot`s to the application's RootFS and finally executes the application.

Here's a comparison of the default and fly stage1:

stage1-coreos.aci:

```
host OS
  └─ rkt
    └─ systemd-nspawn
      └─ systemd
        └─ chroot
          └─ user-app1
```


stage1-fly.aci:

```
host OS
  └─ rkt
    └─ chroot
      └─ user-app1
```

### Mount propagation modes
The *fly* stage1 makes use of Linux' [mount propagation modes](https://www.kernel.org/doc/Documentation/filesystems/sharedsubtree.txt).
If a volume source path is a mountpoint on the host, this mountpoint is made recursively shared before the host path is mounted on the target path in the container.
Hence, changes to the mounts on the target mount path inside the container will be propagated back to the host.

The bind mounts for `/dev`, `/proc`, and `/sys` are done automatically and are recursive, because their hierarchy contains mounts which also need to be available for the container to function properly.
User provided volumes are not mounted recursively.
This is a safety measure to prevent system crashes when multiple containers are started that mount `/` into the container. 


## Getting started

You can either use `stage1-fly.aci` from the official release, or build rkt yourself with the right options:

```
$ ./autogen.sh && ./configure --with-stage1-flavors=fly && make
```

For more details about configure parameters, see [configure script parameters documentation](build-configure.md).
This will build the rkt binary and the stage1-fly.aci in `build-rkt-0.13.0+git/bin/`.

### Selecting stage1 at runtime

Curious readers can read a whole document on how to [choose which stage1.aci to use at runtime](https://github.com/coreos/rkt/blob/master/Documentation/commands.md#use-a-custom-stage-1).

Here is a quick example of how to use a container stage1 named `stage1-fly.aci` in `/usr/local/rkt/`:
```
# rkt run --stage1-image=/usr/local/rkt/stage1-fly.aci coreos.com/etcd:v2.0.9
```


## WARNING: missing isolation and security features

The *fly* stage1 does **NOT** support the isolators and security features as the default stage1 does.

Here's an incomplete list of features that are missing:
- network namespace isolation
- CPU isolators
- Memory isolators
- CAPABILITY bounding
- SELinux

### Winning missing features back with systemd

If you run systemd on your host, you can [wrap rkt with a systemd unit file](using-rkt-with-systemd.md#advanced-unit-file).
For more information please consult the systemd manual. 

The following should get you started:

* [systemd.resource-control](http://www.freedesktop.org/software/systemd/man/systemd.resource-control.html) 
* [systemd.directives](http://www.freedesktop.org/software/systemd/man/systemd.directives.html)

