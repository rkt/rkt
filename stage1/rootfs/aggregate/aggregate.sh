#!/bin/bash -e

# aggregate everything into a single rootfs tree

ROOT=s1rootfs

# always start over
[ -e "$ROOT" ] && rm -Rf "$ROOT"

# run everything in install.d/*
for i in install.d/*; do
	source "$i"
done
