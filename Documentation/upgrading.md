## Upgrading rkt

Usually, upgrading rkt should work without doing anything special: pods started with the old version will continue running and new pods will be started with the new version.
However, in some cases special care must be taken.

### When the rkt api-service is running

If the api-service is running and the new version of rkt does a store version upgrade that requires migration, new invocations of rkt will be blocked.
This is so because the api-service is a long running process that holds a lock on the store, and the store migration needs to take an exclusive lock on it.

For this reason, it is recommended to stop the api-service and start the latest version when upgrading rkt.

This recommendation doesn't apply if the new api-service is listening on a different port and using a different [rkt data directory](commands.md#global-options) via the `--dir` flag.
