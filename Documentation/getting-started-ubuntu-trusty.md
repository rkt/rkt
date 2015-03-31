# Getting Started with rkt on Ubuntu Trusty

The following guide will show you how to build and run the sample [etcd aci](https://github.com/coreos/etcd/releases/download/v2.0.0/etcd-v2.0.0-linux-amd64.aci) on the standard vagrantcloud.com [box for ubuntu trusty](https://vagrantcloud.com/ubuntu/boxes/trusty64).


## Download and start an Ubuntu Trusty box

```
vagrant init ubuntu/trusty64
vagrant up --provider virtualbox
```

## SSH into the VM, Install rkt and Download the ACI

```
vagrant ssh
sudo su

wget https://github.com/coreos/rkt/releases/download/v0.3.1/rkt-v0.3.1.tar.gz
tar xzvf rkt-v0.3.1.tar.gz
cd rkt-v0.3.1
./rkt help
```
## Trust the CoreOS signing key

This shows how to trust the CoreOS signing key with the procedure outlined in [Signing and Verification Guide](https://github.com/coreos/rkt/blob/master/Documentation/signing-and-verification-guide.md)

Download the public key
`curl -O https://coreos.com/dist/pubkeys/aci-pubkeys.gpg`

Get the fingerprint
```
gpg --with-fingerprint aci-pubkeys.gpg
```

```
pub  4096R/88182190 2015-01-23 CoreOS ACI Builder <release@coreos.com>
Key fingerprint = 8B86 DE38 890D DB72 9186  7B02 5210 BD88 8818 2190
```

Format the fingerprint for the filename
```
echo "8B86 DE38 890D DB72 9186  7B02 5210 BD88 8818 2190" | tr -d "[:space:]" | tr '[:upper:]' '[:lower:]'
```

```
8b86de38890ddb7291867b025210bd8888182190
```

Create the directory to trust coreos.com/etcd aci's signed by this key
```
mkdir -p /etc/rkt/trustedkeys/prefix.d/coreos.com/etcd/
```

Move the key to a filename with the fingerprint of the key
```
mv aci-pubkeys.gpg /etc/rkt/trustedkeys/prefix.d/coreos.com/etcd/8b86de38890ddb7291867b025210bd8888182190
```

## Fetch the ACI

```
./rkt fetch https://github.com/coreos/etcd/releases/download/v2.0.0/etcd-v2.0.0-linux-amd64.aci
rkt: fetching image from https://github.com/coreos/etcd/releases/download/v2.0.0/etcd-v2.0.0-linux-amd64.aci
Downloading aci: [                                             ] 16.4 KB/3.58 MB
Downloading signature from https://github.com/coreos/etcd/releases/download/v2.0.0/etcd-v2.0.0-linux-amd64.sig
rkt: signature verified:
  CoreOS ACI Builder <release@coreos.com>
sha512-fcdf12587358af6ebe69b5338a05df67
```

## Run the ACI

Now try running the ACI and you should see output like this:

```
./rkt run https://github.com/coreos/etcd/releases/download/v2.0.0/etcd-v2.0.0-linux-amd64.aci
rkt: fetching image from https://github.com/coreos/etcd/releases/download/v2.0.0/etcd-v2.0.0-linux-amd64.aci
2015/01/25 04:56:34 no data-dir provided, using default data-dir ./default.etcd
2015/01/25 04:56:34 etcd: listening for peers on http://localhost:2380
2015/01/25 04:56:34 etcd: listening for peers on http://localhost:7001
2015/01/25 04:56:34 etcd: listening for client requests on http://localhost:2379
2015/01/25 04:56:34 etcd: listening for client requests on http://localhost:4001
2015/01/25 04:56:34 etcdserver: name = default
2015/01/25 04:56:34 etcdserver: data dir = default.etcd
2015/01/25 04:56:34 etcdserver: heartbeat = 100ms
2015/01/25 04:56:34 etcdserver: election = 1000ms
2015/01/25 04:56:34 etcdserver: snapshot count = 10000
2015/01/25 04:56:34 etcdserver: advertise client URLs = http://localhost:2379,http://localhost:4001
2015/01/25 04:56:34 etcdserver: initial advertise peer URLs = http://localhost:2380,http://localhost:7001
2015/01/25 04:56:34 etcdserver: initial cluster = default=http://localhost:2380,default=http://localhost:7001
2015/01/25 04:56:34 etcdserver: start member ce2a822cea30bfca in cluster 7e27652122e8b2ae
2015/01/25 04:56:34 raft: ce2a822cea30bfca became follower at term 0
2015/01/25 04:56:34 raft: newRaft ce2a822cea30bfca [peers: [], term: 0, commit: 0, applied: 0, lastindex: 0, lastterm: 0]
2015/01/25 04:56:34 raft: ce2a822cea30bfca became follower at term 1
2015/01/25 04:56:34 etcdserver: added local member ce2a822cea30bfca [http://localhost:2380 http://localhost:7001] to cluster 7e27652122e8b2ae
2015/01/25 04:56:35 raft: ce2a822cea30bfca is starting a new election at term 1
2015/01/25 04:56:35 raft: ce2a822cea30bfca became candidate at term 2
2015/01/25 04:56:35 raft: ce2a822cea30bfca received vote from ce2a822cea30bfca at term 2
2015/01/25 04:56:35 raft: ce2a822cea30bfca became leader at term 2
2015/01/25 04:56:35 raft.node: ce2a822cea30bfca elected leader ce2a822cea30bfca at term 2
2015/01/25 04:56:35 etcdserver: published {Name:default ClientURLs:[http://localhost:2379 http://localhost:4001]} to cluster 7e27652122e8b2ae

Press ^] three times to kill container
```

You are now running etcd inside of a rkt pod on Ubuntu Trusty.
