# Fine-Grained Pod Manipulations

To provide and API for imperative application operations (e.g. start|stop|add)
inside a pod for finer-grained control over rkt containerization concepts and
debugging needs, this proposal introduces new stage1 entrypoints and a
subcommand CLI API that will be used for manipulating applications inside pods.

The motivation behind this change is the direction orchestration systems are
taking and to facilitate this new direction. For more details, see
[kubernetes#25899](https://github.com/yujuhong/kubernetes/blob/08dc66113399c89e31f6872f3c638695a6ec6a8d/docs/proposals/container-runtime-interface-v1.md).

# API

The envisioned workflow for the app-level CLI API is that after a pod has been started, 
the users will be invoking the rkt CLI which consist of application logic on top of aforementioned stage1 entrypoints.

## rkt app add
Injects an application image into a running pod.

It first prepares an application rootfs in stage0 for the application image/images
and then the prepared-app is injected via the entrypoint.

```bash
rkt app add <pod-uuid> --app=<app-name> <image-name/hash/address/registry-URL> <arguments>
```

**Note:** Not every pod will be injectable and it will be enabled through an option.

## rkt app rm
Removes an application image into a running pod, meaning that the leftover resources are removed from the pod.

```bash
rkt app rm <pod-uuid> --app=<app-name> <arguments>
```

**Note:** when a pod becomes empty, currently it will terminate, but this
proposal will introduce a `--mutable` or `--allow-empty` or `--dumb` flag to be
used when starting pods, so that the lifecycle management of the pod is left to
the user. Meaning that empty pods won't be terminated.

### Resources Leftover by an Application in Default Stage1 Flavor
- Rootfs (e.g. `/opt/stage/<app-name>`)
- Mounts from volumes (e.g. `/opt/stage/<app-name>/<volume-name>`)
- Mounts related to rkt operations (e.g. `/opt/stage/<app-name>/dev/null`)
- Systemd service files (e.g. `<app-name>.service` and `reaper-<app-name>.service`)
- Miscellaneous files (e.g. `/rkt/<app-name>.env`, `/rkt/status...`)

## rkt app start
Starts an application that was injected.
This operation is idempotent.

```bash
rkt app start <pod-uuid> --app=<app-name> <arguments>
```

## rkt app stop
Stops a running application gracefully (grace is defined in the `app stop` entrypoint section).

```bash
rkt app stop <pod-uuid> --app=<app-name>
```

## rkt app list
Lists the applications that are inside a pod, running or stopped.

```bash
rkt app list <pod-uuid> <arguments>
```

**Note:** List should consist of an app specifier and status at the very least,
the rest is up for discussions.

## rkt app status
Returns the execution status of application inside a pod.

```bash
rkt app status <pod-uuid> --app=<app-name> <arguments>
```

The returned status information for an example mysql application would contain
the following details (output format is up for discussion):

```go
type AppStatus struct {
	Name       string
	State      AppState
	CreatedAt  time.Time
	StartedAt  time.Time
	FinishedAt time.Time
	ExitCode   int64 
}
```

**Note:** status will be obtained from an annotated json file residing in stage1
that contains the required information.

## rkt app exec
Executes a command inside an application.

```bash
rkt app exec <pod-uuid> --app=<app-name> <arguments> -- <command> <command-arguments>
```

# Entrypoints

In order to facilitate the app-level operations API, 4 new entrypoints are introduced.

For example, the `app start` entrypoint in default stage1 flavor sends a `start` signal
(similar to `systemctl start`) for the service files of the application to the systemd which is the stage1.

## rkt app add

`coreos.com/rkt/stage1/app/add`

1. `coreos.com/rkt/stage1/add` entrypoint is resolved via annotations found within `/var/lib/rkt/pods/run/$uuid/stage1/manifest`.
2. The entrypoint will be executed and passed a reference to the runtime-manifest of the app that was prepared.
3. Perform setup based on the runtime-manifest of the app.

The responsibility of this entrypoint is to receive a prepared app, inject it
into the pod, where it will be started using `app/start` entrypoint.

### Stage1 Default Flavor Example Workflow

1. Render the application rootfs.
2. Prepare the application/s
3. Start the injected application using the `app/start` entrypoint.

This approach is similar to using `rkt run`, instead of creating a new pod,
an existing pod will be used.

## rkt app rm

`coreos.com/rkt/stage1/app/rm`

1. receive a reference to an application that resides inside the pod (running or stopped)
2. stop the application if its running.
3. remove the contents of the application (rootfs) from the pod (keep the logs?) and delete references to it (e.g. service files).
   - Delegate these steps by creating `app/install` and `app/rm` entrypoints?

The responsibility of this entrypoint is to receive an app inside a pod and to remove it from the pod, so that it is ready to be started by the stage1 (e.g. systemd for the default flavor).
After `rm`, starting the application again is not possible and requires injecting the same
application using the `app/add` entrypoint.

### Stage1 Default Flavor Example Workflow

1. Stop the application using the `app stop` entrypoint.
2. Collect the garbage left by the app (e.g. rootfs, unit files, etc.)
3. Reload the systemd daemon for changes to take place without disturbing the other tennant applications.

## rkt app start

`coreos.com/rkt/stage1/app/start`

1. given a reference to an application that's in the `Prepared` state, start the application.

The responsibility of this entrypoint is to start an application that is in the `Prepared` state, 
which is an app that was recently injected, instruct the stage1 to start the application.

## rkt app stop

`coreos.com/rkt/stage1/app/stop`

1. given a reference to an application that's in the `Running` state (see below), instruct the stage1 to stop the application.

The responsibility of this entrypoint is to stop an application that is `Running` by instructing the stage1. 

**Note:** there is a graceful shutdown which sends the termination signal to application and waits for a grace period
for application to terminate and then the application is shutdown if it doesn't terminate by the end of the grace period.

# App States

Expected set of app states are listed below:

```go
type AppState string

const (
	UnknownAppState AppState = "unknown"

	PreparingAppState AppState = "preparing"

	// Apps that are ready to be used by `app start`.
	PreparedAppState AppState = "prepared"

	RunningAppState AppState = "running"

	// Apps stopped by `app stop`.
	StoppingAppState AppState = "stopping"

	// Apps that finish their execution naturally.
	ExitedAppState AppState = "exited"

	// Once an app is marked for ejection, while the ejection is being
	// performed, no further operations can be done on that app.
	DeletingAppState AppState = "deleting"
)
```

**Note:** State transitions are linear in that an app that is in state `Exited` cannot transition into `Running` state.

# Use Cases

## Low-level  Pods

Grant granular access to pods for orchestration systems and allow orchestration systems to develop their
own pod concept on top of the exposed app-level operations.

### Workflow

1. Create an empty pod (specified via an option that allows empty pods to stay alive).
2. Inject applications into the pod.
3. Orchestrate the workflow of applications (e.g. app1 has to terminate successfully before app2).

## Pod Mutability

Enable in-place updates of a pod without disrupting the operations of the pod.

### Workflow

1. Remove the old applications without disturbing/restarting the whole pod.
2. Inject updated applications.
3. Start the updated applications.

### Note

Allows the consumers to perform rolling updates inside running pods.

## Debugging Pods

Allow users to inject debug applications into a pod in production.

### Example Workflow

1. Deploy an application containing only a Go web service binary.
2. Encounter an error not decipherable via the available information (e.g. status info, logs, etc.).
3. Inject a debug ACI containing binaries (e.g. `lsof`) for debugging the service.
4. Enter the pod namespace and use the debug binaries.
