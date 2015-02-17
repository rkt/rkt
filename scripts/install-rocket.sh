#!/bin/bash

set -xe
export DEBIAN_FRONTEND=noninteractive

ROCKET_VERSION=0.3.2
APP_SPEC_VERSION=0.3.0

if ! [ -d app-spec ]; then
  echo "Install actool ${APP_SPEC_VERSION}"
  mkdir app-spec
  wget -q -O appc-spec.tar.gz https://github.com/appc/spec/archive/v${APP_SPEC_VERSION}.tar.gz
  tar xzvf appc-spec.tar.gz -C app-spec --strip-components=1
  pushd app-spec
  ./build
  sudo cp -v bin/* /usr/local/bin
  popd
fi

if ! [ -d rocket ]; then
  echo "Install rocket ${ROCKET_VERSION}"
  mkdir rocket

  # install deps
  which unsquashfs || sudo apt-get install -y squashfs-tools

  # grab rocket
  wget -q -O rocket.tar.gz https://github.com/coreos/rocket/archive/v${ROCKET_VERSION}.tar.gz
  tar xzvf rocket.tar.gz -C rocket --strip-components=1

  # build/install rocket
  pushd rocket
  ./build
  sudo cp -v bin/* /usr/local/bin
  popd
fi

