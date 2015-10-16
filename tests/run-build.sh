#!/bin/bash

set -e

# Skip build if requested
if test -e ci-skip ; then
    cat last-commit
    echo
    echo "Build skipped as requested in the last commit."
    exit 0
fi

RKT_STAGE1_USR_FROM="${1}"
RKT_STAGE1_SYSTEMD_VER="${2}"
BUILD_DIR="build-rkt-${RKT_STAGE1_USR_FROM}-${RKT_STAGE1_SYSTEMD_VER}"

if [ "${CI-}" == true ] ; then
	# https://semaphoreci.com/docs/available-environment-variables.html
	if [ "${SEMAPHORE-}" == true ] ; then
        # Setup go environment on semaphore
        if [ -f /opt/change-go-version.sh ]; then
            . /opt/change-go-version.sh
            change-go-version 1.4
        fi

        # Semaphore does not clean git subtrees between each build.
        sudo rm -rf "${BUILD_DIR}"
    fi
fi

mkdir -p builds
cd builds

git clone ../ "${BUILD_DIR}"

cd "${BUILD_DIR}"

./autogen.sh
case "${RKT_STAGE1_USR_FROM}" in
    coreos|kvm)
	./configure --with-stage1-flavors="${RKT_STAGE1_USR_FROM}" \
		    --with-stage1-default-flavor="${RKT_STAGE1_USR_FROM}" \
		    --enable-functional-tests
	;;
    host)
	./configure --with-stage1-flavors=host \
		    --with-default-stage1-flavor=host \
		    --enable-functional-tests=auto
	;;
    src)
	./configure --with-stage1-flavors="${RKT_STAGE1_USR_FROM}" \
		    --with-stage1-default-flavor="${RKT_STAGE1_USR_FROM}" \
		    --with-stage1-systemd-version="${RKT_STAGE1_SYSTEMD_VER}" \
		    --enable-functional-tests
	;;
    none)
	# Not a flavor per se, so perform a detailed setup for some
	# hypothetical 3rd party stage1 image
	./configure --with-stage1-default-name="example.com/some-stage1-for-rkt" \
		    --with-stage1-default-version="0.0.1"
	;;
    *)
	echo "Unknown flavor: ${RKT_STAGE1_USR_FROM}"
	exit 1
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
