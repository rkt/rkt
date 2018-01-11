#!/bin/bash

set -e

DEBIAN_SID_DEPS="ca-certificates gcc libc6-dev gpg gpg-agent make automake wget git golang-go coreutils cpio squashfs-tools autoconf file xz-utils patch bc locales libacl1-dev libssl-dev libtspi-dev libsystemd-dev python pkg-config zlib1g-dev libglib2.0-dev libpixman-1-dev libcap-dev"

export DEBIAN_FRONTEND=noninteractive
apt-get update
apt-get install -y --no-install-recommends ${DEBIAN_SID_DEPS}
