#!/usr/bin/env bash
set -e

echo "Building worker..."
CGO_ENABLED=0 GOOS=linux go build -o worker-binary -a -tags netgo -ldflags '-w' ./log-stresser/main.go

rmWorker() { rm worker-binary; }
acbuildEnd() {
    rmWorker
    export EXIT=$?
    acbuild --debug end && exit $EXIT
}

trap rmWorker EXIT

acbuild --debug begin

trap acbuildEnd EXIT

acbuild --debug set-name appc.io/rkt-log-stresser

acbuild --debug copy worker-binary /worker

acbuild --debug set-exec -- /worker

acbuild --debug write --overwrite log-stresser.aci
