# Inspect how rkt works

There's a variety of ways to inspect how containers work.
Linux provides APIs that expose information about namespaces (proc filesystem) and cgroups (cgroup filesystem).
We also have tools like strace that allow us to see what system calls are used in processes.

This document explains how to use those APIs and tools to give details on what rkt does under the hood.

Note that this is not a comprehensive analysis of the inner workings of rkt, but a starting point for people interested in learning how containers work.

## What syscalls does rkt use?

Let's use [strace][strace] to find out what system calls rkt uses to set up containers.
We'll only trace a handful of syscalls since, by default, strace traces every syscall resulting in a lot of output.
Also, we'll redirect its output to a file to make the analysis easier.

```bash
$ sudo strace -f -s 512 -e unshare,clone,mount,chroot,execve -o out.txt rkt run coreos.com/etcd:v2.0.10
...
^]^]Container rkt-e6d92625-aa3f-4449-bf5d-43ffed440de4 terminated by signal KILL.
```

We now have our trace in `out.txt`, let's go through some of its relevant parts.

### stage0

First, we see the actual execution of the rkt command:

```
5710  execve("/home/iaguis/work/go/src/github.com/rkt/rkt/build-rkt/target/bin/rkt", ["rkt", "run", "coreos.com/etcd:v2.0.10"], 0x7ffce2052be8 /* 22 vars */) = 0
```

Since the image was already fetched and we don't trace many system calls, nothing too exciting happens here except mounting the container filesystems.

```
5710  mount("overlay", "/var/lib/rkt/pods/run/d5513d49-d14f-45d1-944b-39437798ddda/stage1/rootfs", "overlay", 0, "lowerdir=/var/lib/rkt/cas/tree/deps-sha512-cc076d6c508223cc3c13c24d09365d64b6d15e7915a165eab1d9e87f87be5015/rootfs,upperdir=/var/lib/rkt/pods/run/d5513d49-d14f-45d1-944b-39437798ddda/overlay/deps-sha512-cc076d6c508223cc3c13c24d09365d64b6d15e7915a165eab1d9e87f87be5015/upper,workdir=/var/lib/rkt/pods/run/d5513d49-d14f-45d1-944b-39437798ddda/overlay/deps-sha512-cc076d6c508223cc3c13c24d09365d64b6d15e7915a165eab1d9e87f87be5015/work") = 0
5710  mount("overlay", "/var/lib/rkt/pods/run/d5513d49-d14f-45d1-944b-39437798ddda/stage1/rootfs/opt/stage2/etcd/rootfs", "overlay", 0, "lowerdir=/var/lib/rkt/cas/tree/deps-sha512-c0de11e9d504069810da931c94aece3bcc5430dc20f9a5177044eaef62f93fcc/rootfs,upperdir=/var/lib/rkt/pods/run/d5513d49-d14f-45d1-944b-39437798ddda/overlay/deps-sha512-c0de11e9d504069810da931c94aece3bcc5430dc20f9a5177044eaef62f93fcc/upper/etcd,workdir=/var/lib/rkt/pods/run/d5513d49-d14f-45d1-944b-39437798ddda/overlay/deps-sha512-c0de11e9d504069810da931c94aece3bcc5430dc20f9a5177044eaef62f93fcc/work/etcd") = 0
```

We can see that rkt mounts the stage1 and stage2 filesystems with the [tree store][treestore] as `lowerdir` on the directory rkt expects them to be.

Note that the stage2 tree is mounted within the stage1 tree via `/opt/stage2`.
You can read more about the tree structure in [rkt architecture][architecture-stage0].

This means that, for a same tree store, everything will be shared in a copy-on-write (COW) manner, except the bits that each container modifies, which will be in the `upperdir` and will appear magically in the mount destination.
You can read more about this filesystem in the [overlay documentation][overlay].


### stage1

This is where most of the interesting things happen, the first being executing the stage1 [run entrypoint][run-entrypoint], which is `/init` by default:

```
5710  execve("/var/lib/rkt/pods/run/d5513d49-d14f-45d1-944b-39437798ddda/stage1/rootfs/init", ["/var/lib/rkt/pods/run/d5513d49-d14f-45d1-944b-39437798ddda/stage1/rootfs/init", "--net=default", "--local-config=/etc/rkt", "d5513d49-d14f-45d1-944b-39437798ddda"], 0xc42009b040 /* 25 vars */ <unfinished ...>
```

init does a bunch of stuff, including creating the container's network namespace and mounting a reference to it on the host filesystem:

```
5723  unshare(CLONE_NEWNET)             = 0
5723  mount("/proc/5710/task/5723/ns/net", "/var/run/netns/cni-eee014d2-8268-39cc-c176-432bbbc9e959", 0xc42017c6a8, MS_BIND, NULL) = 0
```

After creating the network namespace, it will execute the relevant [CNI][cni] plugins from within that network namespace.
The default network uses the [ptp plugin][ptp] with [host-local][host-local] as IPAM:

```
5725  execve("stage1/rootfs/usr/lib/rkt/plugins/net/ptp", ["stage1/rootfs/usr/lib/rkt/plugins/net/ptp"], 0xc4201ac000 /* 32 vars */ <unfinished ...>
5730  execve("stage1/rootfs/usr/lib/rkt/plugins/net/host-local", ["stage1/rootfs/usr/lib/rkt/plugins/net/host-local"], 0xc42008e240 /* 32 vars */ <unfinished ...>
```

In this case, the CNI plugins use come from rkt's stage1, but [rkt is also able to pick a CNI plugin installed externally](https://github.com/rkt/rkt/blob/v1.29.0/Documentation/networking/overview.md#custom-plugins).

The plugins will do some iptables magic to configure the network:

```
5739  execve("/usr/bin/iptables", ["iptables", "--version"], 0xc42013e000 /* 32 vars */ <unfinished ...>
5740  execve("/usr/bin/iptables", ["/usr/bin/iptables", "-t", "nat", "-N", "CNI-7a59ad232c32bcea94ee08d5", "--wait"], 0xc4200b0a20 /* 32 vars */ <unfinished ...>
...
```

After the network is configured, rkt mounts the container cgroups instead of letting systemd-nspawn do it because we want to have control on how they're mounted.
We also mount the host cgroups if they're not already mounted in the way systemd-nspawn expects them, like in old distributions or distributions that don't use systemd (e.g. [Void Linux][void-linux]).

We do this in a new mount namespace to avoid polluting the host mounts and to get automatic cleanup when the container exits (`CLONE_NEWNS` is the flag for [mount namespaces][man-namespaces] for historical reasons: it was the first namespace implemented on Linux):

```
5710  unshare(CLONE_NEWNS)              = 0
```

Here we mount the container hierarchies read-write so the pod can modify its cgroups but we mount the controllers read-only so the pod doesn't modify other cgroups:

```
5710  mount("stage1/rootfs/sys/fs/cgroup/freezer/machine.slice/machine-rkt\\x2dd5513d49\\x2dd14f\\x2d45d1\\x2d944b\\x2d39437798ddda.scope/system.slice", "stage1/rootfs/sys/fs/cgroup/freezer/machine.slice/machine-rkt\\x2dd5513d49\\x2dd14f\\x2d45d1\\x2d944b\\x2d39437798ddda.scope/system.slice", 0xc42027d2a8, MS_BIND, NULL) = 0
...
5710  mount("stage1/rootfs/sys/fs/cgroup/freezer", "stage1/rootfs/sys/fs/cgroup/freezer", 0xc42027d2b8, MS_RDONLY|MS_NOSUID|MS_NODEV|MS_NOEXEC|MS_REMOUNT|MS_BIND, NULL) = 0
```

Now is the time to start systemd-nspawn to create the pod itself:

```
5710  execve("stage1/rootfs/usr/lib/ld-linux-x86-64.so.2", ["stage1/rootfs/usr/lib/ld-linux-x86-64.so.2", "stage1/rootfs/usr/bin/systemd-nspawn", "--boot", "--notify-ready=yes", "--register=true", "--link-journal=try-guest", "--quiet", "--uuid=d5513d49-d14f-45d1-944b-39437798ddda", "--machine=rkt-d5513d49-d14f-45d1-944b-39437798ddda", "--directory=stage1/rootfs", "--capability=CAP_AUDIT_WRITE,CAP_CHOWN,CAP_DAC_OVERRIDE,CAP_FSETID,CAP_FOWNER,CAP_KILL,CAP_MKNOD,CAP_NET_RAW,CAP_NET_BIND_SERVICE,CAP_SETUID,CAP_SETGID,CAP_SETPCAP,CAP_SETFCAP,CAP_SYS_CHROOT", "--", "--default-standard-output=tty", "--log-target=null", "--show-status=0"], 0xc4202bc0f0 /* 29 vars */ <unfinished ...>
```

Note we don't need to pass the `--private-network` option because rkt already created and configured the network using CNI.

Some interesting things systemd-nspawn does are moving the container filesystem tree to `/`:

```
5747  mount("/var/lib/rkt/pods/run/d5513d49-d14f-45d1-944b-39437798ddda/stage1/rootfs", "/", NULL, MS_MOVE, NULL) = 0
5747  chroot(".")                       = 0
```

And creating all the other namespaces: mount, UTS, IPC, and PID.
Check [namespaces(7)][man-namespaces] for more information.

```
5747  clone(child_stack=NULL, flags=CLONE_NEWNS|CLONE_NEWUTS|CLONE_NEWIPC|CLONE_NEWPID|SIGCHLD) = 5748
```

Once it's done creating the container, it will execute the init process, which is systemd:

```
5748  execve("/usr/lib/systemd/systemd", ["/usr/lib/systemd/systemd", "--default-standard-output=tty", "--log-target=null", "--show-status=0"], 0x7f904604f250 /* 8 vars */) = 0
```

Which then will execute systemd-journald to handle logging:

```
5749  execve("/usr/lib/systemd/systemd-journald", ["/usr/lib/systemd/systemd-journald"], 0x5579c5d79d50 /* 8 vars */ <unfinished ...>
...
```

And at some point it will execute our application's service (in this example, etcd).

But first, it needs to execute its companion `prepare-app` dependency:

```
5751  execve("/prepare-app", ["/prepare-app", "/opt/stage2/etcd/rootfs"], 0x5579c5d7d580 /* 7 vars */) = 0
```

`prepare-app` bind-mounts a lot of files from stage1 to stage2, so our app has access to a [reasonable environment][os-spec]:

```
5751  mount("/dev/null", "/opt/stage2/etcd/rootfs/dev/null", 0x49006f, MS_BIND, NULL) = 0
5751  mount("/dev/zero", "/opt/stage2/etcd/rootfs/dev/zero", 0x49006f, MS_BIND, NULL) = 0
...
```

After it's finished, our etcd service is ready to start!

Since we use some additional security directives in our service file (like [`InaccessiblePaths=`][inaccessible-paths] or [`SystemCallFilter=`][syscall-filter]), systemd will create an additional mount namespace per application in the pod and move the stage2 filesystem to `/`:

```
5753  unshare(CLONE_NEWNS)              = 0
...
5753  mount("/opt/stage2/etcd/rootfs", "/", NULL, MS_MOVE, NULL) = 0
5753  chroot(".")                       = 0
```

### stage2

Now we're ready to execute the etcd binary.

```
5753  execve("/etcd", ["/etcd"], 0x5579c5dbd660 /* 9 vars */) = 0
```

And that's it, etcd is running in a container!

## Inspect running containers with procfs

We'll now inspect a running container by using the [proc filesystem][procfs].

Let's start a new container limiting the CPU to 200 millicores and the memory to 100MB:

```
$ sudo rkt run --interactive kinvolk.io/aci/busybox --memory=100M --cpu=200m
/ # 
```

First we'll need to find the PID of a process running inside the container.
We can see the container PID by running `rkt status`:

```
$ sudo rkt status 567264dd
state=running
created=2018-01-03 17:17:39.653 +0100 CET
started=2018-01-03 17:17:39.749 +0100 CET
networks=default:ip4=172.16.28.37
pid=10985
exited=false
```

Now we need to find the sh process running inside the container:

```
$ ps auxf | grep [1]0985 -A 3
root     10985  0.0  0.0  54204  5040 pts/2    S+   17:17   0:00          \_ stage1/rootfs/usr/lib/ld-linux-x86-64.so.2 stage1/rootfs/usr/bin/systemd-nspawn --boot --notify-ready=yes --register=true --link-journal=try-guest --quiet --uuid=567264dd-f28d-42fb-84a1-4714dde9e82c --machine=rkt-567264dd-f28d-42fb-84a1-4714dde9e82c --directory=stage1/rootfs --capability=CAP_AUDIT_WRITE,CAP_CHOWN,CAP_DAC_OVERRIDE,CAP_FSETID,CAP_FOWNER,CAP_KILL,CAP_MKNOD,CAP_NET_RAW,CAP_NET_BIND_SERVICE,CAP_SETUID,CAP_SETGID,CAP_SETPCAP,CAP_SETFCAP,CAP_SYS_CHROOT -- --default-standard-output=tty --log-target=null --show-status=0
root     11021  0.0  0.0  62280  7392 ?        Ss   17:17   0:00              \_ /usr/lib/systemd/systemd --default-standard-output=tty --log-target=null --show-status=0
root     11022  0.0  0.0  66408  8812 ?        Ss   17:17   0:00                  \_ /usr/lib/systemd/systemd-journald
root     11026  0.0  0.0   1212     4 pts/0    Ss+  17:17   0:00                  \_ /bin/sh
```

It's 11026!

Let's start by having a look at its namespaces:

```
$ sudo ls -l /proc/11026/ns/
total 0
lrwxrwxrwx 1 root root 0 Jan  3 17:19 cgroup -> 'cgroup:[4026531835]'
lrwxrwxrwx 1 root root 0 Jan  3 17:19 ipc -> 'ipc:[4026532764]'
lrwxrwxrwx 1 root root 0 Jan  3 17:19 mnt -> 'mnt:[4026532761]'
lrwxrwxrwx 1 root root 0 Jan  3 17:19 net -> 'net:[4026532702]'
lrwxrwxrwx 1 root root 0 Jan  3 17:19 pid -> 'pid:[4026532765]'
lrwxrwxrwx 1 root root 0 Jan  3 17:19 pid_for_children -> 'pid:[4026532765]'
lrwxrwxrwx 1 root root 0 Jan  3 17:19 user -> 'user:[4026531837]'
lrwxrwxrwx 1 root root 0 Jan  3 17:19 uts -> 'uts:[4026532763]'
```

We can compare it with the namespaces on the host

```
$ sudo ls -l /proc/1/ns/
total 0
lrwxrwxrwx 1 root root 0 Jan  3 17:20 cgroup -> 'cgroup:[4026531835]'
lrwxrwxrwx 1 root root 0 Jan  3 17:20 ipc -> 'ipc:[4026531839]'
lrwxrwxrwx 1 root root 0 Jan  3 17:20 mnt -> 'mnt:[4026531840]'
lrwxrwxrwx 1 root root 0 Jan  3 17:20 net -> 'net:[4026532009]'
lrwxrwxrwx 1 root root 0 Jan  3 17:20 pid -> 'pid:[4026531836]'
lrwxrwxrwx 1 root root 0 Jan  3 17:20 pid_for_children -> 'pid:[4026531836]'
lrwxrwxrwx 1 root root 0 Jan  3 17:20 user -> 'user:[4026531837]'
lrwxrwxrwx 1 root root 0 Jan  3 17:20 uts -> 'uts:[4026531838]'
```

We can see that the cgroup and user namespace are the same, since rkt doesn't use cgroup namespaces and user namespaces weren't enabled for this execution.
If, for example, we run rkt with `--net=host`, we'll see that the network namespace is the same as the host's.

Running [lsns][man-lsns] we can see this information too, along with the PID that created the namespace:

```
> sudo lsns -p 11026
        NS TYPE   NPROCS    PID USER COMMAND
4026531835 cgroup    231      1 root /sbin/init
4026531837 user      231      1 root /sbin/init
4026532702 net         4  10945 root stage1/rootfs/usr/lib/ld-linux-x86-64.so.2 stage1/rootfs/usr/bin/systemd-nspawn --boot --notify-ready=yes --register=true --link-journal=
4026532761 mnt         1  11026 root /etcd
4026532763 uts         3  11021 root /usr/lib/systemd/systemd --default-standard-output=tty --log-target=null --show-status=0
4026532764 ipc         3  11021 root /usr/lib/systemd/systemd --default-standard-output=tty --log-target=null --show-status=0
4026532765 pid         3  11021 root /usr/lib/systemd/systemd --default-standard-output=tty --log-target=null --show-status=0
```

We can also see some interesting data about the process:

```
$ sudo cat /proc/11026/status
Name:	sh
Umask:	0022
State:	S (sleeping)
...
CapBnd:	00000000a80425fb
...
NoNewPrivs:	0
Seccomp:	2
...
```

This tells us the container is not using the `no_new_privs` feature, but it is using [seccomp][seccomp].

We can also see what [capabilities][capabilities] are in the bounding set of the process, let's decode them with `capsh`:

```
$ capsh --decode=00000000a80425fb
0x00000000a80425fb=cap_chown,cap_dac_override,cap_fowner,cap_fsetid,cap_kill,cap_setgid,cap_setuid,cap_setpcap,cap_net_bind_service,cap_net_raw,cap_sys_chroot,cap_mknod,cap_audit_write,cap_setfcap
```

Another interesting thing is the environment variables of the process:

```
$ sudo cat /proc/11026/environ | tr '\0' '\n'
PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin
HOME=/root
LOGNAME=root
USER=root
SHELL=/bin/sh
INVOCATION_ID=d5d94569d482495c809c113fca55abd4
TERM=xterm
AC_APP_NAME=busybox
```

Finally, we can check in which cgroups the process is:

```
$ sudo cat /proc/11026/cgroup
11:freezer:/
10:rdma:/
9:cpu,cpuacct:/machine.slice/machine-rkt\x2d97910fdc\x2d13ec\x2d4025\x2d8f93\x2d5ddea0089eff.scope/system.slice/busybox.service
8:devices:/machine.slice/machine-rkt\x2d97910fdc\x2d13ec\x2d4025\x2d8f93\x2d5ddea0089eff.scope/system.slice/busybox.service
7:blkio:/machine.slice/machine-rkt\x2d97910fdc\x2d13ec\x2d4025\x2d8f93\x2d5ddea0089eff.scope/system.slice
6:memory:/machine.slice/machine-rkt\x2d97910fdc\x2d13ec\x2d4025\x2d8f93\x2d5ddea0089eff.scope/system.slice/busybox.service
5:perf_event:/
4:pids:/machine.slice/machine-rkt\x2d97910fdc\x2d13ec\x2d4025\x2d8f93\x2d5ddea0089eff.scope/system.slice/busybox.service
3:net_cls,net_prio:/
2:cpuset:/
1:name=systemd:/machine.slice/machine-rkt\x2d97910fdc\x2d13ec\x2d4025\x2d8f93\x2d5ddea0089eff.scope/system.slice/busybox.service
0::/machine.slice/machine-rkt\x2d97910fdc\x2d13ec\x2d4025\x2d8f93\x2d5ddea0089eff.scope/system.slice/busybox.service
```

Let's explore the cgroups a bit more.

## Inspect container cgroups

systemd offers tools to easily inspect the cgroups of containers.

We can use `systemd-cgls` to see the cgroup hierarchy of a container:

```
$ machinectl
MACHINE                                  CLASS     SERVICE OS VERSION ADDRESSES
rkt-97910fdc-13ec-4025-8f93-5ddea0089eff container rkt     -  -       172.16.28.25...

1 machines listed.
$ systemd-cgls -M rkt-97910fdc-13ec-4025-8f93-5ddea0089eff
Control group /machine.slice/machine-rkt\x2d97910fdc\x2d13ec\x2d4025\x2d8f93\x2d5ddea0089eff.scope:
-.slice
├─init.scope
│ └─12474 /usr/lib/systemd/systemd --default-standard-output=tty --log-target=null --show-status=0
└─system.slice
  ├─busybox.service
  │ └─12479 /bin/sh
  └─systemd-journald.service
    └─12475 /usr/lib/systemd/systemd-journald
```

And we can use `systemd-cgtop` to see the resource consumption of the container.
This is the output while running the `yes` command (which is basically an infinite loop that outputs the character `y`, so it takes all the CPU) in the container:

```
Control Group                                                                                            Tasks   %CPU   Memory  Input/s Output/s
/machine.slice/machine-rkt\x2d97910fdc\x2d13ec\x2d4025\x2d8f93\x2d5ddea0089eff.scope                         4   19.6     7.6M        -        -
/machine.slice/machine-rkt\x2d97910fdc\x2d13ec\x2d4025\x2d8f93\x2d5ddea0089eff.scope/system.slice            3   19.6     5.1M        -        -
/machine.slice/machine-rkt\x2d979…c\x2d4025\x2d8f93\x2d5ddea0089eff.scope/system.slice/busybox.service       2   19.6   476.0K        -        -
/machine.slice/machine-rkt\x2d979…d8f93\x2d5ddea0089eff.scope/system.slice/system-prepare\x2dapp.slice       -      -   120.0K        -        -
/machine.slice/machine-rkt\x2d979…\x2d8f93\x2d5ddea0089eff.scope/system.slice/systemd-journald.service       1      -     4.5M        -        -
```

You can see that our CPU limit is working, since we only see a CPU usage of about 20%.

This information can also be gathered from the cgroup filesystem itself.
For example, to see the memory consumed by the busybox application:

```
$ /sys/fs/cgroup/memory/machine.slice/machine-rkt\\x2d97910fdc\\x2d13ec\\x2d4025\\x2d8f93\\x2d5ddea0089eff.scope/system.slice/busybox.service/
$ cat memory.usage_in_bytes
487424
```

You can find out more about cgroups in their [kernel documentation][cgroups].

[strace]: https://linux.die.net/man/1/strace
[overlay]: https://www.kernel.org/doc/Documentation/filesystems/overlayfs.txt
[treestore]: ../../store/treestore/tree.go
[run-entrypoint]: stage1-implementors-guide.md#rkt-run
[cni]: https://github.com/containernetworking/cni
[ptp]: ../networking/overview.md#ptp
[host-local]: ../networking/overview.md#host-local
[seccomp]: ../seccomp-guide.md
[procfs]: https://www.kernel.org/doc/Documentation/filesystems/proc.txt
[cgroups]: https://www.kernel.org/doc/Documentation/cgroup-v1/cgroups.txt
[man-namespaces]: http://man7.org/linux/man-pages/man7/namespaces.7.html
[architecture-stage0]: architecture.md#stage-0
[inaccessible-paths]: https://www.freedesktop.org/software/systemd/man/systemd.exec.html#ReadWritePaths=
[syscall-filter]: https://www.freedesktop.org/software/systemd/man/systemd.exec.html#SystemCallFilter=
[capabilities]: ../capabilities-guide.md
[os-spec]: https://github.com/appc/spec/blob/v0.8.11/spec/OS-SPEC.md
[man-lsns]: http://man7.org/linux/man-pages/man8/lsns.8.html
[void-linux]: https://www.voidlinux.eu/
