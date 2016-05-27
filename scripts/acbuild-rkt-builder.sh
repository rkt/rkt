#!/usr/bin/env bash

if [[ $EUID -ne 0 ]]; then
   echo "This script must be run as root" 1>&2
   exit 1
fi

set -ex

function check_tool {
if ! which $1; then
    echo "Get $1 and put it in your \$PATH" >&2;
    exit 1;
fi
}

MODIFY=${MODIFY:-""}
FLAGS=${FLAGS:-""}
IMG_NAME="coreos.com/rkt/builder"
IMG_VERSION=${VERSION:-"v1.7.0"}
ACI_FILE=rkt-builder.aci
BUILDDIR=/opt/build-rkt
SRC_DIR=/opt/rkt
ACI_GOPATH=/go

DEBIAN_SID_DEPS="ca-certificates gcc libc6-dev make automake wget git golang-go cpio squashfs-tools realpath autoconf file xz-utils patch bc locales libacl1-dev libssl-dev libsystemd-dev"

acbuildend () {
    export EXIT=$?;
    acbuild --debug end && rm -rf rootfs && exit $EXIT;
}

echo "Generating debian sid tree"

mkdir rootfs
debootstrap --force-check-gpg --variant=minbase --components=main --include="${DEBIAN_SID_DEPS}" sid rootfs http://httpredir.debian.org/debian/
rm -rf rootfs/var/cache/apt/archives/*

echo "Version: ${IMG_VERSION}"
echo "Building ${ACI_FILE}"

acbuild begin ./rootfs
trap acbuildend EXIT

acbuild $FLAGS set-name $IMG_NAME
acbuild $FLAGS label add version $IMG_VERSION
acbuild $FLAGS set-user 0
acbuild $FLAGS set-group 0
acbuild $FLAGS environment add OS_VERSION sid
acbuild $FLAGS environment add GOPATH $ACI_GOPATH
acbuild $FLAGS environment add BUILDDIR $BUILDDIR
acbuild $FLAGS environment add SRC_DIR $SRC_DIR
acbuild $FLAGS mount add build-dir $BUILDDIR
acbuild $FLAGS mount add src-dir $SRC_DIR
acbuild $FLAGS set-working-dir $SRC_DIR
acbuild $FLAGS copy "$(dirname $0)" /scripts
acbuild $FLAGS run /bin/bash /scripts/install-appc-spec.sh
acbuild $FLAGS set-exec /bin/bash /scripts/build-rkt.sh
if [ -z "$MODIFY" ]; then
    acbuild write --overwrite $ACI_FILE
fi
