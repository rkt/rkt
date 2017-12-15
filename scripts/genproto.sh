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

if ! [[ $(protoc --version) =~ "3.1.0" ]]; then
	echo "could not find protoc 3.1.0, is it installed + in PATH?"
	exit 255
fi

export PATH=.:${PATH}
cd $GOPATH/src/github.com/rkt/rkt

echo "building protoc-gen-go"
pushd vendor/github.com/golang/protobuf/protoc-gen-go
go build
mv protoc-gen-go $(dirs -l +1)
popd

trap 'rm -f "protoc-gen-go"' EXIT

echo "generating code"
API_DIR="${PWD}/api/v1alpha"
protoc -I "${API_DIR}" "${API_DIR}"/*.proto --go_out=plugins=grpc:"${API_DIR}"
echo "generating HTML documentation"
sudo rkt \
    --insecure-options=image \
    run \
    --volume=volume-out,kind=host,source="${API_DIR}"/docs \
    --volume=volume-protos,kind=host,source="${API_DIR}" \
    docker://pseudomuto/protoc-gen-doc:1.0.0 \
    --user=$UID \
    --group=$GID
echo "generating Markdown documentation"
sudo rkt \
    --insecure-options=image \
    run \
    --volume=volume-out,kind=host,source="${API_DIR}"/docs \
    --volume=volume-protos,kind=host,source="${API_DIR}" \
    docker://pseudomuto/protoc-gen-doc:1.0.0 \
    --user=$UID \
    --group=$GID \
    --exec=/entrypoint.sh \
    -- \
    --doc_opt=markdown,docs.md
