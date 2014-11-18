#!/bin/bash -e

# Derive a minimal rootfs for hosting systemd from a coreos release pxe image

URL="http://stable.release.core-os.net/amd64-usr/current/coreos_production_pxe_image.cpio.gz"

function req() {
	what=$1

	which "${what}" >/dev/null || { echo "${what} required"; exit 1; }
}

req unsquashfs
req curl
req gzip
req cpio
req install

# extract the squashfs from the cpio then use unsquashfs to extract the required files for systemd
WORK="mkroot"
USRFS="usr.squashfs"
ROOT="rootfs"
USR="rootfs/usr"
FILELIST="filelist.txt"
DEFAULT_TGT="default.target"
SHUTDOWN_SVC="shutdown.service"

mkdir -p "${WORK}/${ROOT}"

# extract $USR from $URL
pushd "${WORK}"
curl "${URL}" | gzip -cd | cpio --extract "${USRFS}"

# bin/bash and the first group of libs are unnecessary but convenient for nsenter use
cat > "${FILELIST}" << EOF
bin/bash
lib64/libreadline.so
lib64/libreadline.so.6
lib64/libreadline.so.6.2
lib64/libncurses.so
lib64/libncurses.so.5
lib64/libncurses.so.5.9
lib64/libdl.so
lib64/libdl.so.2
lib64/libdl-2.17.so
bin/journalctl
bin/systemctl
bin/systemd-analyze
bin/systemd-ask-password
bin/systemd-cat
bin/systemd-cgls
bin/systemd-cgtop
bin/systemd-coredumpctl
bin/coredumpctl
bin/systemd-delta
bin/systemd-detect-virt
bin/systemd-inhibit
bin/systemd-machine-id-setup
bin/systemd-notify
bin/systemd-nspawn
bin/systemd-path
bin/systemd-run
bin/systemd-stdio-bridge
bin/systemd-sysusers
bin/systemd-tmpfiles
bin/systemd-tty-ask-password-agent
lib
lib64/libattr.so
lib64/libitm.so
lib64/libitm.so.1
lib64/libitm.so.1.0.0
lib64/libstdc++.so
lib64/libstdc++.so.6
lib64/libstdc++.so.6.0.17
lib64/libgcc_s.so
lib64/libgcc_s.so.1
lib64/librt-2.17.so
lib64/libz.so.1
lib64/libc.so
lib64/libz.so.1.2.8
lib64/libattr.so.1.1.0
lib64/libpthread.so.0
lib64/libz.so
lib64/libseccomp.so.2.1.1
lib64/libseccomp.so
lib64/libpthread.so
lib64/libcap.so.2.22
lib64/libpthread-2.17.so
lib64/libkmod.so.2
lib64/ld-linux-x86-64.so.2
lib64/ld-2.17.so
lib64/librt.so.1
lib64/libkmod.so
lib64/libcap.so
lib64/libc-2.17.so
lib64/librt.so
lib64/libseccomp.so.2
lib64/libattr.so.1
lib64/libkmod.so.2.2.5
lib64/libcap.so.2
lib64/libc.so.6
lib64/systemd/systemd-backlight
lib64/systemd/systemd-update-utmp
lib64/systemd/systemd-vconsole-setup
lib64/systemd/systemd-journal-remote
lib64/systemd/systemd-modules-load
lib64/systemd/systemd-resolved
lib64/systemd/systemd-bus-proxyd
lib64/systemd/systemd-ac-power
lib64/systemd/systemd-bootchart
lib64/systemd/systemd-initctl
lib64/systemd/systemd-shutdown
lib64/systemd/systemd-multi-seat-x
lib64/systemd/systemd-rfkill
lib64/systemd/systemd-networkd
lib64/systemd/systemd-activate
lib64/systemd/systemd-readahead
lib64/systemd/systemd-hostnamed
lib64/systemd/systemd-random-seed
lib64/systemd/systemd-cgroups-agent
lib64/systemd/systemd-udevd
lib64/systemd/systemd-shutdownd
lib64/systemd/systemd-logind
lib64/systemd/systemd
lib64/systemd/systemd-update-done
lib64/systemd/systemd-machined
lib64/systemd/systemd-user-sessions
lib64/systemd/systemd-sysctl
lib64/systemd/systemd-journald
lib64/systemd/systemd-timedated
lib64/systemd/systemd-networkd-wait-online
lib64/systemd/systemd-localed
lib64/systemd/systemd-cryptsetup
lib64/systemd/user-generators
lib64/systemd/systemd-remount-fs
lib64/systemd/systemd-coredump
lib64/systemd/systemd-timesyncd
lib64/systemd/systemd-socket-proxyd
lib64/systemd/system-shutdown
lib64/systemd/systemd-binfmt
lib64/systemd/systemd-fsck
lib64/systemd/system-sleep
lib64/systemd/systemd-sleep
lib64/systemd/systemd-reply-password
lib64/systemd/systemd-journal-gatewayd
EOF

unsquashfs -d "${USR}" -ef "${FILELIST}" "${USRFS}" 

popd

# install static stage2 target service file
install -D -m 644 "${DEFAULT_TGT}" "${WORK}/${ROOT}/usr/lib/systemd/system/${DEFAULT_TGT}"
install -D -m 644 "${SHUTDOWN_SVC}" "${WORK}/${ROOT}/usr/lib/systemd/system/${SHUTDOWN_SVC}"
install -d "${WORK}/${ROOT}/usr/lib/systemd/system/${DEFAULT_TGT}.wants"

install -d "${WORK}/${ROOT}/etc"
echo "rocket" > "${WORK}/${ROOT}/etc/os-release"

# parent dir for the stage2 bind mounts
install -d "${WORK}/${ROOT}/opt/stage2"

# install /init
install -m 755 init "${WORK}/${ROOT}"
