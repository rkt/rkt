# Running rkt with an LKVM stage1

rkt has experimental support for running with an [LKVM](https://kernel.googlesource.com/pub/scm/linux/kernel/git/will/kvmtool/+/master/README) stage1.
That is, rkt will start a virtual machine with full hypervisor isolation instead of creating a container using Linux cgroups and namespaces.

## Getting started

To use a LKVM stage1 you need to build rkt with the right options:

```
$ ./autogen.sh && ./configure --with-stage1=kvm && make
```

This will build the rkt binary and the LKVM stage1.aci in `build-rkt-0.7.0/bin/`.

Provided you have hardware virtualization support and the [kernel KVM module](http://www.linux-kvm.org/page/Getting_the_kvm_kernel_modules) loaded (refer to your distribution for instructions), you can then run an image like you would normally do with rkt:

```
# rkt trust --prefix coreos.com/etcd
Prefix: "coreos.com/etcd"
Key: "https://coreos.com/dist/pubkeys/aci-pubkeys.gpg"
GPG key fingerprint is: 8B86 DE38 890D DB72 9186  7B02 5210 BD88 8818 2190
        CoreOS ACI Builder <release@coreos.com>
Are you sure you want to trust this key (yes/no)?
yes
Trusting "https://coreos.com/dist/pubkeys/aci-pubkeys.gpg" for prefix "coreos.com/etcd".
Added key for prefix "coreos.com/etcd" at "/etc/rkt/trustedkeys/prefix.d/coreos.com/etcd/8b86de38890ddb7291867b025210bd8888182190"
# rkt run coreos.com/etcd:v2.0.9
rkt: searching for app image coreos.com/etcd:v2.0.9
Downloading signature from https://github.com/coreos/etcd/releases/download/v2.0.9/etcd-v2.0.9-linux-amd64.aci.asc
Downloading signature: [=======================================] 819 B/819 B
Downloading ACI: [=============================================] 3.79 MB/3.79 MB
rkt: signature verified:
  CoreOS ACI Builder <release@coreos.com>
2015/08/06 11:06:01 Preparing stage1
2015/08/06 11:06:02 Loading image sha512-91e98d7f1679a097c878203c9659f2a26ae394656b3147963324c61fa3832f15
2015/08/06 11:06:02 Writing pod manifest
2015/08/06 11:06:02 Setting up stage1
2015/08/06 11:06:02 Writing image manifest
2015/08/06 11:06:02 Wrote filesystem to /var/lib/rkt/pods/run/8570ff05-4050-4864-9ec3-eb23471ff0e5
2015/08/06 11:06:02 Writing image manifest
2015/08/06 11:06:02 Pivoting to filesystem /var/lib/rkt/pods/run/8570ff05-4050-4864-9ec3-eb23471ff0e5
2015/08/06 11:06:02 Execing /init
[    0.243979] etcd[71]: 2015/08/06 09:06:02 etcd: no data-dir provided, using default data-dir ./default.etcd
[    0.245979] etcd[71]: 2015/08/06 09:06:02 etcd: listening for peers on http://localhost:2380
[    0.245979] etcd[71]: 2015/08/06 09:06:02 etcd: listening for peers on http://localhost:7001
[    0.245979] etcd[71]: 2015/08/06 09:06:02 etcd: listening for client requests on http://localhost:2379
[    0.246978] etcd[71]: 2015/08/06 09:06:02 etcd: listening for client requests on http://localhost:4001
[    0.246978] etcd[71]: 2015/08/06 09:06:02 etcdserver: datadir is valid for the 2.0.1 format
[    0.247978] etcd[71]: 2015/08/06 09:06:02 etcdserver: name = default
[    0.247978] etcd[71]: 2015/08/06 09:06:02 etcdserver: data dir = default.etcd
[    0.247978] etcd[71]: 2015/08/06 09:06:02 etcdserver: member dir = default.etcd/member
[    0.247978] etcd[71]: 2015/08/06 09:06:02 etcdserver: heartbeat = 100ms
[    0.247978] etcd[71]: 2015/08/06 09:06:02 etcdserver: election = 1000ms
[    0.248978] etcd[71]: 2015/08/06 09:06:02 etcdserver: snapshot count = 10000
[    0.248978] etcd[71]: 2015/08/06 09:06:02 etcdserver: advertise client URLs = http://localhost:2379,http://localhost:4001
[    0.248978] etcd[71]: 2015/08/06 09:06:02 etcdserver: initial advertise peer URLs = http://localhost:2380,http://localhost:7001
[    0.248978] etcd[71]: 2015/08/06 09:06:02 etcdserver: initial cluster = default=http://localhost:2380,default=http://localhost:7001
[    0.256977] etcd[71]: 2015/08/06 09:06:02 etcdserver: start member ce2a822cea30bfca in cluster 7e27652122e8b2ae
[    0.256977] etcd[71]: 2015/08/06 09:06:02 raft: ce2a822cea30bfca became follower at term 0
[    0.257977] etcd[71]: 2015/08/06 09:06:02 raft: newRaft ce2a822cea30bfca [peers: [], term: 0, commit: 0, applied: 0, lastindex: 0, lastterm: 0]
[    0.257977] etcd[71]: 2015/08/06 09:06:02 raft: ce2a822cea30bfca became follower at term 1
[    0.264976] etcd[71]: 2015/08/06 09:06:02 etcdserver: added local member ce2a822cea30bfca [http://localhost:2380 http://localhost:7001] to cluster 7e27652122e8b2ae
[    1.559048] etcd[71]: 2015/08/06 09:06:03 raft: ce2a822cea30bfca is starting a new election at term 1
[    1.560272] etcd[71]: 2015/08/06 09:06:03 raft: ce2a822cea30bfca became candidate at term 2
[    1.561807] etcd[71]: 2015/08/06 09:06:03 raft: ce2a822cea30bfca received vote from ce2a822cea30bfca at term 2
[    1.562889] etcd[71]: 2015/08/06 09:06:03 raft: ce2a822cea30bfca became leader at term 2
[    1.563920] etcd[71]: 2015/08/06 09:06:03 raft.node: ce2a822cea30bfca elected leader ce2a822cea30bfca at term 2
[    1.569898] etcd[71]: 2015/08/06 09:06:03 etcdserver: published {Name:default ClientURLs:[http://localhost:2379 http://localhost:4001]} to cluster 7e27652122e8b2ae
```

This output is the same you'll get if you run a container-based rkt.
If you want to see the kernel and boot messages, run rkt with the `--debug` flag.

You can exit pressing `<Ctrl-a x>`.

### Selecting stage1 at runtime

If you want to run software that requires hypervisor isolation along with trusted software that only needs container isolation, you can [choose which stage1.aci to use at runtime](https://github.com/coreos/rkt/blob/master/Documentation/commands.md#use-a-custom-stage-1).

For example, if you have a container stage1 named `stage1-container.aci` and a lkvm stage1 named `stage1-lkvm.aci` in `/usr/local/rkt/`:

```
# rkt run --stage1-image=/usr/local/rkt/stage1-container.aci coreos.com/etcd:v2.0.9
...
# rkt run --stage1-image=/usr/local/rkt/stage1-lkvm.aci coreos.com/etcd:v2.0.9
...
```

## What doesn't work

LKVM support is in an very early stage so not every feature works yet:

* Network support [#1219](https://github.com/coreos/rkt/pull/1219)

## How does it work?

It leverages the work done by Intel with their [Clear Containers system](https://lwn.net/Articles/644675/).
Stage1 contains a Linux kernel that is executed under LKVM.
This kernel will then start systemd, which in turn will start the applications in the pod.

A LKVM-based rkt is very similar to a container-based one, it just uses lkvm to execute pods instead of systemd-nspawn.

Here's a comparison of the components involved between a container-based and a LKVM based rkt.

Container-based:

```
host OS
  └─ rkt
    └─ systemd-nspawn
      └─ systemd
        └─ chroot
          └─ user-app1
```


LKVM based:

```
host OS
  └─ rkt
    └─ lkvm
      └─ kernel
        └─ systemd
          └─ chroot
            └─ user-app1
```
