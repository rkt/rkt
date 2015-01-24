# Life-cycle of a container in rocket

## Creation

`rkt run` instantiates a container UUID and file-system tree rooted at `/var/lib/rkt/containers/$uuid/.`, henceforth referred to as `$cdir`.

A robust mechanism is necessary for reliably determining whether a container is executing so garbage collection may be safely and reliably implemented.  In lieu of a daemon process responsible for managing the execution of containers, an advisory lock associated with an open file descriptor in the container's /init process context is used instead.  The advisory lock is acquired exclusively on `$cdir` early in the container execution, this lock is automatically released by the kernel when the /init process exits.

## Death / exit

A container is considered exited if an advisory lock can be acquired on `$cdir`.  Upon exit of a container's /init process, the exclusive `$cdir` lock acquired at the beginning of `rkt run` becomes released by the kernel.

## Garbage collection

Exited containers are discarded using a common mark-and-sweep style of garbage collection by invoking the `rkt gc` command.  This relatively simple approach lends itself well to a minimal file-system-based implementation utilizing no additional daemons or record-keeping with good efficiency.  The process is performed in two distinct passes explained below in detail.

### Pass 1: mark

All directories found in `/var/lib/rkt/containers` are tested for exited status by trying to acquire a shared advisory lock on each directory.

When a directory's lock cannot be acquired, the directory is skipped as it indicates the container is currently executing.

For directories which the lock is successfully acquired, the directory is atomically renamed from `/var/lib/rkt/containers/$uuid` to `/var/lib/rkt/garbage/$uuid`.  The renaming effectively implements the "mark" operation.  The locks are immediately released, and operations like `rkt status` may occur simultaneous to `rkt gc`.

Containers dwell in the `/var/lib/rkt/garbage` directory for a grace-period during which their status may continue to be queried by `rkt status`.  The rename from `/var/lib/rkt/containers/$uuid` to `/var/lib/rkt/garbage/$uuid` serves to keep exited containers from cluttering the `/var/lib/rkt/containers` directory during their respective dwell periods.

### Pass 2: sweep

A side-effect of the rename operation responsible for moving a container from `/var/lib/rkt/containers` to `/var/lib/rkt/garbage` is an update to the container directory's change time.  The sweep operation takes advantage of this in honoring the necessary grace-period before discarding exited containers.  This grace period currently defaults to 30 minutes, and may be explicitly specified using the `--grace-period duration` option of `rkt gc`.  Note that this grace period begins from the time a container was marked by `rkt gc`, not when the container exited.  A container becomes eligible for marking upon exit, but will not be marked until a subsequent `rkt gc` is performed.

The change times of all directories found in `/var/lib/rkt/garbage` are compared against the current time.  Directories having sufficiently old change times are locked exclusively and recursively deleted.  If a lock acquisition fails, the directory is skipped.  Failed exclusive lock acquisitions may occur if the garbage container is currently being accessed via `rkt status`, for example.  The skipped containers will be revisited on a subsequent `rkt gc` invocation's sweep pass.

## Pulse

To answer the questions "Has this container exited?" and "Is this container being deleted?" the container's UUID is looked for in `/var/lib/rkt/containers` and `/var/lib/rkt/garbage`.  Containers found in the `/var/lib/rkt/garbage` directory must already be exited, and a shared lock acquisition may be used to determine if the garbage container is actively being deleted.  Those found in the `/var/lib/rkt/containers` directory may be exited or running, a failed shared lock acquisition indicates a container in `/var/lib/rkt/containers` is alive at the time of the failed acquisition.
