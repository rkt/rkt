# Security best practices

This document tries to give an overview of the security recommendations to follow when running rkt containers and discuss non-obvious communication channels between the host and the pod.

## General recommendations

* **Don't run applications as root.** Use the [user and group fields][aci-schema] in your images or the `--user` and `--group` [CLI flags][rkt-run-subcommands] when running containers.
* **Don't disable security features unless you really need it.** You should not use the `--insecure-options=` flag unless strictly necessary.
* **Restrict capabilities given to your images as much as possible.** Only those actually needed by your app [should be granted][capabilities-guide].
* **Tweak seccomp syscall filtering instead of disabling it.** The default seccomp profile might be too restrictive for your app. If that's the case, [tweak the seccomp profile][seccomp-guide] instead of disabling it.
* **If you're not affected by its [current limitations][user-ns-limitations], use user namespaces.**
* **Don't use host networking** since that will give the app in the container access to the host network interfaces and allow it to connect to any other application listening on the host, including on abstract Unix sockets.

## Volumes

When using volumes, special care should be taken to avoid dangerous interactions with the host.
Here are some security best practices:

* **Use read-only volumes unless writing to them is necessary.**
* If you use Linux v4.2 or older, **avoid sharing directories when tools on the host can move files outside the directory** (such as Nautilus moving directories to the trash bin when a user deletes it). This could expose the entire host filesystem to the container. See [moby/moby#12317 (comment)](https://github.com/moby/moby/issues/12317#issuecomment-92692061).
* To avoid the previous point: **share a full filesystem instead of just a directory in a filesystem if possible**. For example, a mounted partition or some file mounted with `mount -o loop`.
* **Sharing devices from the host to the container is generally not recommended**. If you need to do it, you can find examples in the [block devices documentation](block-devices.md).

## Incoming mounts

Mounts can be added to a pod at runtime by using [`machinectl bind`][machinectl-bind] or when adding apps to a rkt pod with the [app experiment][app-experiment].

This could be used in a malicious way although it requires privileges on the host.

## File descriptors

In certain situations, some file descriptors belonging to the host namespaces are sent to the pod.

### Socket activation

rkt supports [socket-activated services][rkt-socket-activated] and [socket-proxyd][socket-proxyd].
This means some file descriptors will be passed from the host to the container when using those features.

In particular, all file descriptors defined in the socket unit file accompanying the service file where rkt is started are passed to the container.

### sd\_notify

rkt supports the [sd_notify][sd_notify] mechanism to notify systemd on the host that the pod is ready.
This means systemd in the pod will receive a file descriptor which can be used to send information about whether the apps are initialized or not to the host.

[aci-schema]: https://github.com/appc/spec/blob/master/spec/aci.md#image-manifest-schema
[rkt-run-subcommands]: subcommands/run.md#options
[capabilities-guide]: capabilities-guide.md
[seccomp-guide]: seccomp-guide.md
[user-ns-limitations]: devel/user-namespaces.md#current-limitations
[rkt-socket-activated]: using-rkt-with-systemd.md#socket-activated-services
[socket-proxyd]: using-rkt-with-systemd.md#bidirectionally-proxy-local-sockets-to-another-possibly-remote-socket
[sd_notify]: https://www.freedesktop.org/software/systemd/man/sd_notify.html
[machinectl-bind]: https://www.freedesktop.org/software/systemd/man/machinectl.html#bind%20NAME%20PATH%20%5BPATH%5D
[app-experiment]: devel/pod-lifecycle.md#app
