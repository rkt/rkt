#!/usr/bin/env bash
set -e

if [ "$EUID" -ne 0 ]; then
    echo "This script uses functionality which requires root privileges"
    exit 1
fi

# Start the build with an empty ACI
acbuild --debug begin docker://ubuntu:17.10

# In the event of the script exiting, end the build
trap "{ export EXT=$?; acbuild --debug end && exit $EXT; }" EXIT

# Name the ACI
acbuild --debug set-name example.com/play-example

# Install java, psql, and git
acbuild --debug run -- apt-get update
acbuild --debug run -- apt-get -y install openjdk-8-jre openjdk-8-jdk git postgresql

# Clone play example repo
acbuild --debug run -- git clone https://github.com/ics-software-engineering/play-example-postgresql.git /play-example

acbuild --debug run -- apt-get -y autoremove --purge git

acbuild --debug run -- chown -R postgres:postgres /play-example

STAGE_DIR=/play-example/target/universal/stage/
TARGET_BIN="${STAGE_DIR}"/bin/play-example-postgresql

# Build example
acbuild --debug run -- /bin/bash -c 'cd /play-example && su postgres -c "./activator stage"'

# Set user and group
acbuild --debug set-user postgres
acbuild --debug set-group postgres

# Copy "wait for postgres" script
acbuild --debug copy wait-for-postgres.sh /wait-for-postgres.sh

# Run postgres server
acbuild --debug set-exec /wait-for-postgres.sh 127.0.0.1 "${TARGET_BIN}"

# Write the result
acbuild --debug write --overwrite play-latest-linux-amd64.aci
