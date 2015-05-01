# Getting Started with rkt on Ubuntu Trusty

The following guide will show you how to build and run the sample [etcd ACI](https://github.com/coreos/etcd/releases/download/v2.0.9/etcd-v2.0.9-linux-amd64.aci) on the standard vagrantcloud.com [box for Ubuntu Trusty](https://vagrantcloud.com/ubuntu/boxes/trusty64).


## Download and start an Ubuntu Trusty box

```
vagrant init ubuntu/trusty64
vagrant up --provider virtualbox
```

## SSH into the VM and Install rkt

```
vagrant ssh
sudo su

wget https://github.com/coreos/rkt/releases/download/v0.5.5/rkt-v0.5.5.tar.gz
tar xzvf rkt-v0.5.5.tar.gz
cd rkt-v0.5.5
./rkt help
```

## Trust the CoreOS signing key

This shows how to trust the CoreOS signing key using the [`rkt trust` command](commands.md#rkt-trust). 

```
./rkt  trust  --prefix coreos.com/etcd
Prefix: "coreos.com/etcd"
Key: "https://coreos.com/dist/pubkeys/aci-pubkeys.gpg"
GPG key fingerprint is: 8B86 DE38 890D DB72 9186  7B02 5210 BD88 8818 2190
	CoreOS ACI Builder <release@coreos.com>
	Are you sure you want to trust this key (yes/no)? yes
	Trusting "https://coreos.com/dist/pubkeys/aci-pubkeys.gpg" for prefix "coreos.com/etcd".
	Added key for prefix "coreos.com/etcd" at "/etc/rkt/trustedkeys/prefix.d/coreos.com/etcd/8b86de38890ddb7291867b025210bd8888182190"
```

For more details on how signature verification works in rkt, see the [Signing and Verification Guide](https://github.com/coreos/rkt/blob/master/Documentation/signing-and-verification-guide.md).

## Fetch the ACI

The simplest way to retrieve the etcd ACI is to use image discovery:

```
./rkt fetch coreos.com/etcd:v2.0.9 
rkt: searching for app image coreos.com/etcd:v2.0.9
rkt: fetching image from https://github.com/coreos/etcd/releases/download/v2.0.9/etcd-v2.0.9-linux-amd64.aci
Downloading signature from https://github.com/coreos/etcd/releases/download/v2.0.9/etcd-v2.0.9-linux-amd64.aci.asc
Downloading ACI: [================================             ] 2.71 MB/3.79 MB
rkt: signature verified: 
  CoreOS ACI Builder <release@coreos.com>
  sha512-91e98d7f1679a097c878203c9659f2a2
```

For more on this and other ways to retrieve ACIs, check out the `rkt fetch` section of the [commands guide](commands.md#rkt-fetch).


## Run the ACI

Finally, let's run the application we just retrieved:

```
./rkt run coreos.com/etcd:v2.0.9
  rkt: searching for app image coreos.com/etcd:v2.0.9
  rkt: fetching image from https://github.com/coreos/etcd/releases/download/v2.0.9/etcd-v2.0.9-linux-amd64.aci
  2015/04/19 03:57:32 etcd: no data-dir provided, using default data-dir ./default.etcd
  2015/04/19 03:57:32 etcd: listening for peers on http://localhost:2380
  2015/04/19 03:57:32 etcd: listening for peers on http://localhost:7001
  2015/04/19 03:57:32 etcd: listening for client requests on http://localhost:2379
  2015/04/19 03:57:32 etcd: listening for client requests on http://localhost:4001
  2015/04/19 03:57:32 etcdserver: datadir is valid for the 2.0.1 format
  2015/04/19 03:57:32 etcdserver: name = default
  2015/04/19 03:57:32 etcdserver: data dir = default.etcd
  2015/04/19 03:57:32 etcdserver: member dir = default.etcd/member
  2015/04/19 03:57:32 etcdserver: heartbeat = 100ms
  2015/04/19 03:57:32 etcdserver: election = 1000ms
  2015/04/19 03:57:32 etcdserver: snapshot count = 10000
  2015/04/19 03:57:32 etcdserver: advertise client URLs = http://localhost:2379,http://localhost:4001
  2015/04/19 03:57:32 etcdserver: initial advertise peer URLs = http://localhost:2380,http://localhost:7001
  2015/04/19 03:57:32 etcdserver: initial cluster = default=http://localhost:2380,default=http://localhost:7001
  2015/04/19 03:57:32 etcdserver: start member ce2a822cea30bfca in cluster 7e27652122e8b2ae
  2015/04/19 03:57:32 raft: ce2a822cea30bfca became follower at term 0
  2015/04/19 03:57:32 raft: newRaft ce2a822cea30bfca [peers: [], term: 0, commit: 0, applied: 0, lastindex: 0, lastterm: 0]
  2015/04/19 03:57:32 raft: ce2a822cea30bfca became follower at term 1
  2015/04/19 03:57:32 etcdserver: added local member ce2a822cea30bfca [http://localhost:2380 http://localhost:7001] to cluster 7e27652122e8b2ae
  2015/04/19 03:57:33 raft: ce2a822cea30bfca is starting a new election at term 1
  2015/04/19 03:57:33 raft: ce2a822cea30bfca became candidate at term 2
  2015/04/19 03:57:33 raft: ce2a822cea30bfca received vote from ce2a822cea30bfca at term 2
  2015/04/19 03:57:33 raft: ce2a822cea30bfca became leader at term 2
  2015/04/19 03:57:33 raft.node: ce2a822cea30bfca elected leader ce2a822cea30bfca at term 2
  2015/04/19 03:57:33 etcdserver: published {Name:default ClientURLs:[http://localhost:2379 http://localhost:4001]} to cluster 7e27652122e8b2ae
```

Congratulations! You've run your first application with rkt.
For more on how to use rkt, check out the [commands guide](commands.md).
