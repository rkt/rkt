# Debugging rkt

This guide gives an overview of the debugging techniques that can be used to pinpoint problems with rkt itself or with particular images.

## rkt enter

[enter](../subcommands/enter.md) is the most straight-forward technique.
It allows you to enter the namespaces of an app within a pod so you can explore its filesystem and see what's running.

By default it tries to run `/bin/bash` if it's present in the app's filesystem but you can specify a command to run if the app doesn't provide a bash binary.

```bash
$ rkt list
UUID		APP	IMAGE NAME			STATE	CREATED		STARTED		NETWORKS
54ca5d22	busybox	kinvolk.io/aci/busybox:1.24	running	1 hour ago	1 hour ago	default:ip4=172.16.28.114
$ sudo rkt enter 54ca5d22 sh
/ # ls
bin   dev   etc   home  proc  root  run   sys   tmp   usr   var
/ # ps
PID   USER     TIME   COMMAND
    1 root       0:00 /usr/lib/systemd/systemd --default-standard-output=tty --log-target=null --show-status=0
    3 root       0:00 /usr/lib/systemd/systemd-journald
    6 root       0:00 /bin/sh
   40 root       0:00 sh
   44 root       0:00 ps
```

This works well for images that have a shell baked in but a common practice is creating minimal images with only a statically compiled binary or a binary plus its needed libraries, which limits the usefulness of `rkt enter`.
We'll later see how to overcome this limitation.

## Entering stage1

As explained in [rkt architecture](architecture.md), execution with rkt is divided into several distinct stages.
While with `rkt enter` you enter stage2, it is sometimes useful to see what happens in stage1.
For example, you might want to check what options are passed to the unit file of an app, or interact with stage1 systemd.

rkt doesn't include a subcommand you can use to enter stage1, but you can use [`nsenter`](http://man7.org/linux/man-pages/man1/nsenter.1.html) for that purpose.
Note that this only applies for container stage1 images: kvm pods run in an isolated VM and `nsenter` can't enter its stage1, and fly only allows one app per pod and it doesn't have long-running processes on stage1 so there's nothing to enter.

Let's enter a pod's stage1.
We'll need to find stage1 systemd's PID, which is the child of the PID shown in `rkt status`.
Then we can just call nsenter with it:

```bash
$ rkt status 86e6df38
state=running
created=2017-12-19 14:49:00.376 +0100 CET
started=2017-12-19 14:49:00.466 +0100 CET
networks=default:ip4=172.16.28.27
pid=8469
exited=false
$ ps auxf | grep -A1 [8]469
root      8469  0.1  0.0  54204  5076 pts/1    S+   14:48   0:00          \_ stage1/rootfs/usr/lib/ld-linux-x86-64.so.2 stage1/rootfs/usr/bin/systemd-nspawn --boot --notify-ready=yes --register=true --link-journal=try-guest --quiet --uuid=86e6df38-0762-4261-af21-d2b265555179 --machine=rkt-86e6df38-0762-4261-af21-d2b265555179 --directory=stage1/rootfs --capability=CAP_AUDIT_WRITE,CAP_CHOWN,CAP_DAC_OVERRIDE,CAP_FSETID,CAP_FOWNER,CAP_KILL,CAP_MKNOD,CAP_NET_RAW,CAP_NET_BIND_SERVICE,CAP_SETUID,CAP_SETGID,CAP_SETPCAP,CAP_SETFCAP,CAP_SYS_CHROOT -- --default-standard-output=tty --log-target=null --show-status=0
root      8505  0.0  0.0  62276  7300 ?        Ss   14:48   0:00              \_ /usr/lib/systemd/systemd --default-standard-output=tty --log-target=null --show-status=0
$ sudo nsenter -m -u -i -p -t 8505
-bash-4.3#
```

However, the stage1 environment doesn't include basic binaries necessary for debugging:

```bash
-bash-4.3# ls
-bash: ls: command not found
-bash-4.3# cat
-bash: cat: command not found
```

If you only need to inspect the stage1 filesystem, you can check `/proc/$PID/root/` from the host:

```bash
# ls /proc/27107/root/
app_add  app_start  attach  dev		enter	   etc	   gc	 iottymux  lib64  prepare-app  reaper.sh  root	stop  systemd-version  usr
app_rm	 app_stop   bin     diagnostic	enterexec  flavor  init  lib	   opt	  proc	       rkt	  run	sys   tmp	       var
```

Luckily, rkt includes a helper script that injects a static busybox binary and makes entering stage1 easier.
It can be found in `scripts/debug/stage1_install_busybox.sh`.
It takes a pod UUID as a parameter and it outputs the right nsenter command:

```bash
$ rkt list --full
UUID					APP	IMAGE NAME			IMAGE ID		STATE	CREATED					STARTED			NETWORKS
86e6df38-0762-4261-af21-d2b265555179	busybox	kinvolk.io/aci/busybox:1.24	sha512-140375b2a2bd	running	2017-12-19 14:49:00.376 +0100 CET	2017-12-19 14:49:00.466 +0100 CET	default:ip4=172.16.28.27
$ ./scripts/debug/stage1_install_busybox.sh 86e6df38-0762-4261-af21-d2b265555179
Busybox installed. Use the following command to enter pod's stage1:
sudo nsenter -m -u -i -p -t 8505
-bash-4.3# cat rkt/env/busybox
SHELL=/bin/sh
USER=root
LOGNAME=root
HOME=/root
PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin
AC_APP_NAME=busybox
```

By default, the script creates links for common busybox programs, but if you need to run more commands you can call busybox accordingly:

```bash
-bash-4.3# busybox head -n2 /lib/systemd/system/busybox.service
[Unit]
After=systemd-journald.service
```

Run `busybox` without arguments to see a list of available commands.

If you need custom tools you can add them by copying them to `/proc/$PID/root/bin`.
It's easier if they're static binaries to avoid complications with dynamic libraries.

## Debugging images

As mentioned, `rkt enter` has limitations when the image to debug doesn't ship with tools to debug it or even a shell.
One possible way to debug those images is to use the ACI dependencies mechanism to add a custom tools image to your container.

Let's use [actool](https://github.com/appc/spec/tree/v0.8.11/actool) to patch the manifest.

```bash
$ rkt fetch debug-image.aci
sha512-847f2ca60e473121311fac62d014a4f1
$ actool cat-manifest --pretty-print image-to-patch.aci > manifest.json
```

We'll now add the dependency to the manifest, right before the "app" section:

```json
    ],
    "dependencies": [
        {
            "imageID": "sha512-847f2ca60e473121311fac62d014a4f1"
        }
    ],
    "app": {
```

Then we'll patch the image and run it:

```bash
$ actool patch-manifest --manifest manifest.json --replace image-to-patch.aci
$ sudo rkt --insecure-options=image run --interactive image-to-patch.aci
(...)
```

You'll have your favorite debugging tools available.

For examples of how to build images, check out [Build container examples](../build-container-examples).
Creating images with statically built binaries is preferred to avoid complications with shared libraries.

As an alternative to adding a dependency in the ACI, you can also add a volume after a container has been started using [`machinectl bind`](https://www.freedesktop.org/software/systemd/man/machinectl.html#bind%20NAME%20PATH%20%5BPATH%5D):

```bash
$ sudo mkdir /proc/$PID/root/opt/stage2/$APP_NAME/rootfs/tools
$ machinectl
MACHINE                                  CLASS     SERVICE OS VERSION ADDRESSES
rkt-b7458bf5-8e19-40c1-a7dc-7c5aa25e4970 container rkt     -  -       172.16.28.27...

1 machines listed.
$ sudo machinectl bind $PWD/tools /opt/stage2/$APP_NAME/rootfs/tools
```

Where `$PID` is still the PID of systemd in the pod.

## strace

[Strace](http://man7.org/linux/man-pages/man1/strace.1.html) is a very powerful tool to inspect what applications are doing.
For container stage1 images or fly, you can just attach strace to processes running inside pods like any other process on the host:

Let's trace the shell process running in our example busybox container.
We'll also follow forks to trace new processes created by the shell:

```bash
$ ps auxf | grep -A3 [8]469
root      8469  0.1  0.0  54204  5076 pts/1    S+   14:48   0:00          \_ stage1/rootfs/usr/lib/ld-linux-x86-64.so.2 stage1/rootfs/usr/bin/systemd-nspawn --boot --notify-ready=yes --register=true --link-journal=try-guest --quiet --uuid=86e6df38-0762-4261-af21-d2b265555179 --machine=rkt-86e6df38-0762-4261-af21-d2b265555179 --directory=stage1/rootfs --capability=CAP_AUDIT_WRITE,CAP_CHOWN,CAP_DAC_OVERRIDE,CAP_FSETID,CAP_FOWNER,CAP_KILL,CAP_MKNOD,CAP_NET_RAW,CAP_NET_BIND_SERVICE,CAP_SETUID,CAP_SETGID,CAP_SETPCAP,CAP_SETFCAP,CAP_SYS_CHROOT -- --default-standard-output=tty --log-target=null --show-status=0
root      8505  0.0  0.0  62276  7300 ?        Ss   14:48   0:00              \_ /usr/lib/systemd/systemd --default-standard-output=tty --log-target=null --show-status=0
root      8507  0.0  0.0  66408  8740 ?        Ss   14:48   0:00                  \_ /usr/lib/systemd/systemd-journald
root      8510  0.0  0.0   1212   672 pts/3    Ss+  14:48   0:00                  \_ /bin/sh
$ sudo strace -f -p 8510
wait4(-1, [{WIFEXITED(s) && WEXITSTATUS(s) == 0}], WSTOPPED, NULL) = 2646
--- SIGCHLD {si_signo=SIGCHLD, si_code=CLD_EXITED, si_pid=2646, si_uid=0, si_status=0, si_utime=0, si_stime=0} ---
ioctl(10, TIOCSPGRP, [6])               = 0
wait4(-1, 0x7ffc0226b918, WNOHANG|WSTOPPED, NULL) = -1 ECHILD (No child processes)
wait4(-1, 0x7ffc0226b7b8, WNOHANG|WSTOPPED, NULL) = -1 ECHILD (No child processes)
write(1, "This is the current date: ", 26) = 26
clone(strace: Process 12807 attached
(...)
```

To find out more about strace, check out Julia Evans' [strace zine](https://jvns.ca/zines/#strace-zine).

## Debugging applications with gdb

Sometimes strace is not enough to figure out problems with your applications.
From version 7.10, it is possible to use [gdb](https://www.gnu.org/software/gdb/) to debug processes running in containers from the host.
Same as with strace, you need to find the PID of the process you want to debug and then just run:

```
$ sudo gdb -p $PID
```
