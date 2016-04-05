#!/usr/bin/env bash
#
# Generate rkt protobuf bindings.
# Run from repository root.
#
set -e

if ! [[ "$0" =~ "scripts/genproto.sh" ]]; then
	echo "must be run from repository root"
	exit 255
fi

if ! [[ $(protoc --version) =~ "3.0.0" ]]; then
	echo "could not find protoc 3.0.0, is it installed + in PATH?"
	exit 255
fi

export GOPATH=$(mktemp -d)
export PATH=${GOPATH}/bin:${PATH}

trap 'rm -rf "${GOPATH}"' EXIT

# git (sha) version of golang/protobuf
GO_PROTOBUF_SHA="dda510ac0fd43b39770f22ac6260eb91d377bce3"

echo "installing golang/protobuf using GOPATH=${GOPATH}"
go get -u github.com/golang/protobuf/{proto,protoc-gen-go}

echo "resetting golang/protobuf to version ${GO_PROTOBUF_SHA}"
pushd ${GOPATH}/src/github.com/golang/protobuf
	git reset --hard "${GO_PROTOBUF_SHA}"
	make install
popd

API_DIR="api/v1alpha"
protoc -I "${API_DIR}" "${API_DIR}"/*.proto --go_out=plugins=grpc:"${API_DIR}"
