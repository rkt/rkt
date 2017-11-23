## Using block devices in rkt

Using block devices in containers is usually restricted via the [device cgroup controller](https://www.kernel.org/doc/Documentation/cgroup-v1/devices.txt).
This is done to prevent users from doing dangerous things like creating a physical disk device via `mknod` and writing data to it.

```
# ls -l /dev/nvme0n1p1
brw-rw---- 1 root disk 259, 1 Nov  3 15:13 /dev/nvme0n1p1
# rkt run --interactive kinvolk.io/aci/busybox
/ # mknod /dev/nvme0n1p1 b 259 1
mknod: /dev/nvme0n1p1: Operation not permitted
```

When accessing devices from inside the container is actually desired, users can set up rkt volumes and mounts.
In that case, rkt will automatically configure the device cgroup controller for the container without the aforementioned restriction.

```
# rkt run --volume disk,kind=host,source=/dev/nvme0n1p1,readOnly=true \
          --interactive \
          kinvolk.io/aci/busybox \
          --mount volume=disk,target=/dev/nvme0n1p1
/ # ls -l /dev/nvme0n1p1
brw-rw----    1 root     disk      259,   1 Nov  3 14:13 /dev/nvme0n1p1
/ # head /dev/nvme0n1p1 -c11
�X�mkfs.fat/ # 
/ # echo 1 > /dev/nvme0n1p1
/bin/sh: can't create /dev/nvme0n1p1: Operation not permitted
```

Note that the volume is read-only, so we can't write to it because rkt sets a read-only policy in the device cgroup.

For completeness, users can also use `--insecure-options=paths`, which disables any block device protection.
Then, users can just create devices with `mknod`:

```
# rkt run --insecure-options=paths \
          --interactive \
          kinvolk.io/aci/busybox
/ # mknod /dev/nvme0n1p1 b 259 1
/ # ls -l /dev/nvme0n1p1
brw-r--r--    1 root     root      259,   1 Nov  3 15:43 /dev/nvme0n1p1
```

## Examples

Here are some real-world examples that use block devices.

### FUSE + SSHFS

SSHFS allows mounting remote directories over ssh.
In this example we'll mount a remote directory on `/mnt` inside the container.
For this to work, we need to be able to mount and umount filesystems inside the container so we pass the appropriate seccomp and capability options:

```
# rkt run --insecure-options=image \
          --dns=8.8.8.8 \
          --interactive \
          --volume fuse,kind=host,source=/dev/fuse \
          docker://ubuntu \
          --mount volume=fuse,target=/dev/fuse \
          --seccomp mode=retain,@rkt/default-whitelist,mount,umount2 \
          --caps-retain=CAP_SETUID,CAP_SETGID,CAP_DAC_OVERRIDE,CAP_CHOWN,CAP_FOWNER,CAP_SYS_ADMIN
root@rkt-f2098164-b207-41d0-b62b-745659725aee:/# apt-get update && apt-get install sshfs
[...]
root@rkt-f2098164-b207-41d0-b62b-745659725aee:/# sshfs user@host.com: /mnt
The authenticity of host 'host.com (12.34.56.78)' can't be established.
ECDSA key fingerprint is SHA256:L1/2LPI1J6/YlDzbvH+/SF5gamNusPDSqnCSmaNlolc.
Are you sure you want to continue connecting (yes/no)? yes
user@host.com's password: 
root@rkt-f2098164-b207-41d0-b62b-745659725aee:/# cat /mnt/remote-file.txt
HELLO FROM REMOTE
root@rkt-f2098164-b207-41d0-b62b-745659725aee:/# fusermount -u /mnt/
```

### NVIDIA CUDA

CUDA allows using GPUs for general purpose processing and it needs access to the gpu devices.
In this example we also mount the CUDA SDK binaries and the host libraries, and we do some substitution magic to have appc-compliant volume names:

```
# rkt run --insecure-options=image \
          $(for f in /dev/nvidia* /opt/bin/nvidia* /usr/lib/; \
                do echo "--volume $(basename $f | sed 's/\./-/g'),source=$f,kind=host \
                         --mount volume=$(basename $f | sed 's/\./-/g'),target=$f"; \
                done) \
          docker://nvidia/cuda:latest \
          --exec=/opt/bin/nvidia-smi
Wed Sep  7 21:25:22 2016
+-----------------------------------------------------------------------------+
| NVIDIA-SMI 367.35                 Driver Version: 367.35                    |
|-------------------------------+----------------------+----------------------+
| GPU  Name        Persistence-M| Bus-Id        Disp.A | Volatile Uncorr. ECC |
| Fan  Temp  Perf  Pwr:Usage/Cap|         Memory-Usage | GPU-Util  Compute M. |
|===============================+======================+======================|
|   0  GeForce GTX 780     Off  | 0000:01:00.0     N/A |                  N/A |
| 33%   61C    P2    N/A /  N/A |    474MiB /  3018MiB |     N/A      Default |
+-------------------------------+----------------------+----------------------+

+-----------------------------------------------------------------------------+
| Processes:                                                       GPU Memory |
|  GPU       PID  Type  Process name                               Usage      |
|=============================================================================|
|    0                  Not Supported                                         |
+-----------------------------------------------------------------------------+
```

### Format /dev/sdX

You can mount a disk block device (for example, an external USB stick) and format it inside a container.
Like before, if you want to mount it inside the container, you need to pass the appropriate seccomp and capability options:

```
# rkt run --insecure-options=image \
          --volume disk,kind=host,source=/dev/sda,readOnly=false \
          --interactive \
          docker://ubuntu \
          --mount volume=disk,target=/dev/sda
root@rkt-72bd9a93-2e89-4515-8b46-44e0e11c4c79:/# mkfs.ext4 /dev/sda
mke2fs 1.42.13 (17-May-2015)
/dev/sda contains a ext4 file system
	last mounted on Fri Nov  3 17:15:56 2017
Proceed anyway? (y,n) y
Creating filesystem with 491520 4k blocks and 122880 inodes
Filesystem UUID: 9ede01b1-e35b-46a0-b224-24e879973582
Superblock backups stored on blocks:
	32768, 98304, 163840, 229376, 294912

Allocating group tables: done
Writing inode tables: done
Creating journal (8192 blocks): done
Writing superblocks and filesystem accounting information: done

root@rkt-72bd9a93-2e89-4515-8b46-44e0e11c4c79:/# mount /dev/sda /mnt/
root@rkt-72bd9a93-2e89-4515-8b46-44e0e11c4c79:/# echo HELLO > /mnt/hi.txt
root@rkt-72bd9a93-2e89-4515-8b46-44e0e11c4c79:/# cat /mnt/hi.txt
HELLO
root@rkt-72bd9a93-2e89-4515-8b46-44e0e11c4c79:/# umount /mnt/
```
