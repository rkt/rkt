#!/usr/bin/env bash
set -e

if [ "$EUID" -ne 0 ]; then
    echo "This script uses functionality which requires root privileges"
    exit 1
fi

PGDATA=/var/lib/postgresql/data

# Start the build with an empty ACI
acbuild --debug begin

# In the event of the script exiting, end the build
trap "{ export EXT=$?; acbuild --debug end && exit $EXT; }" EXIT

# Name the ACI
acbuild --debug set-name example.com/postgres

# Based on alpine
acbuild --debug dep add quay.io/coreos/alpine-sh

# Install postgres and bash
acbuild --debug run -- apk update
acbuild --debug run -- apk add postgresql bash

# Create postgres data directory
acbuild --debug run -- mkdir -p $PGDATA
acbuild --debug run -- chown -R postgres:postgres $PGDATA

# Create postgres run directory
acbuild --debug run -- mkdir -p /var/run/postgresql
acbuild --debug run -- chown -R postgres:postgres /var/run/postgresql

# Add a mount point for postgres data
acbuild --debug mount add data $PGDATA

# Add a mount point for custom initialization data
acbuild --debug mount add custom-sql /customize.sql

# Set PGDATA env variable
acbuild --debug environment add PGDATA $PGDATA

# Set user and group
acbuild --debug set-user postgres
acbuild --debug set-group postgres

# Set postgres user, group, and test-db
acbuild --debug environment add POSTGRES_USER rkt
acbuild --debug environment add POSTGRES_PASSWORD rkt
acbuild --debug environment add POSTGRES_DB rkt

# Add pre-start hook that will set up the database
acbuild --debug copy postgres-prestart.sh /root/postgres-prestart.sh
acbuild --debug set-event-handler pre-start /root/postgres-prestart.sh postgres

# Add postgres port
acbuild --debug port add postgres tcp 5432

# Run postgres server
acbuild --debug set-exec postgres

# Write the result
acbuild --debug write --overwrite postgres-latest-linux-amd64.aci
