#!/bin/bash

set -e

RKT_STAGE1_USR_FROM="${1}"
RKT_STAGE1_SYSTEMD_VER="${2}"

BUILD_DIR="build-rkt-${RKT_STAGE1_USR_FROM}-${RKT_STAGE1_SYSTEMD_VER}"

mkdir -p builds
cd builds

# Semaphore does not clean git subtrees between each build.
sudo rm -rf "${BUILD_DIR}"

git clone ../ "${BUILD_DIR}"

cd "${BUILD_DIR}"

./autogen.sh
case "${RKT_STAGE1_USR_FROM}" in
    coreos|kvm)
	./configure --with-stage1="${RKT_STAGE1_USR_FROM}" \
		    --enable-functional-tests
	;;
    host|usr-from-host)
	./configure --with-stage1=host \
		    --enable-functional-tests=auto
	;;
    src)
	./configure --with-stage1="${RKT_STAGE1_USR_FROM}" \
		    --with-stage1-systemd-version="${RKT_STAGE1_SYSTEMD_VER}" \
		    --enable-functional-tests
	;;
    *)
	./configure --with-stage1="${RKT_STAGE1_USR_FROM}"
	;;
esac

CORES=$(grep -c ^processor /proc/cpuinfo)
echo "Running make with ${CORES} threads"
make "-j${CORES}"
make check
make "-j${CORES}" clean
cd ..

# Make sure there is enough disk space for the next build
sudo rm -rf "${BUILD_DIR}"
