# Hacking Guide

## Building rocket

### Requirements

* Linux 3.8+
  * make
  * gcc
  * glibc development and static pieces (on Fedora/RHEL/Centos: glibc-devel and glibc-static packages, on Debian/Ubuntu libc6-dev package)
  * cpio
  * squashfs-tools
  * realpath
  * gpg
* Go 1.3+
  * github.com/appc/spec (not yet vendored as it's in a continuous improvement phase)

Once the requirements have been met you can build rocket by running the following commands:

```
git clone https://github.com/coreos/rocket.git
cd rocket; ./build
```

### With Docker

Alternatively, you can build rocket in a docker container with the following command. Replace $SRC with the absolute path to your rocket source code:

```
$ sudo docker run -v $SRC:/opt/rocket -i -t golang:1.3 /bin/bash -c "apt-get update && apt-get install -y coreutils cpio squashfs-tools realpath && cd /opt/rocket && go get github.com/appc/spec/... && ./build"
```
