# Hacking Guide

## Building rocket

### Requirements

* Linux 3.8+
  * cpio
  * squashfs-tools
  * realpath
* Go 1.3+
  * jteeuwen/go-bindata

Once the requirements have been met you can build rocket by running the following commands:

```
git clone https://github.com/coreos/rocket.git
cd rocket; ./build
```

### With Docker

Alternatively, you can build rocket in a docker container with the following command. Replace $SRC with the absolute path to your rocket source code:

```
docker run -v $SRC:/opt/rocket -i -t google/golang /bin/bash -c "apt-get install -y cpio squashfs-tools realpath && cd /opt/rocket && go get github.com/jteeuwen/go-bindata/... && ./build"
```
