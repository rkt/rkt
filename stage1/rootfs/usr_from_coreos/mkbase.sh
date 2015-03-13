#!/bin/bash -e

# Derive a minimal base tree for hosting systemd from a cached coreos release pxe image

shopt -s nullglob

function req() {
	what=$1

	which "${what}" >/dev/null || { echo "${what} required"; exit 1; }
}

req cpio
req gzip
req unsquashfs
req sort

CACHED_IMG="cache/pxe.img"
SQUASH="usr.squashfs"
ROOTFS="rootfs"
USR="rootfs/usr"
FILELIST="manifest.txt"

# always start with an empty rootfs
[ -e "${ROOTFS}" ] && rm -Rf "${ROOTFS}"

mkdir -p "${ROOTFS}"

# create consolidated file list
cat manifest.d/* | sort -u > "${FILELIST}"

# derive $SQUASH from $CACHED_IMG
gzip -cd "${CACHED_IMG}" | cpio --unconditional --extract "${SQUASH}"
unsquashfs -d "${USR}" -ef "${FILELIST}" "${SQUASH}"

# just leave the desired tree @ ${ROOTFS}
