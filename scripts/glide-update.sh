#!/usr/bin/env bash
#
# Update vendored dedendencies.
#
set -e

if ! [[ "$0" =~ "scripts/glide-update.sh" ]]; then
	echo "must be run from repository root"
	exit 255
fi

if [ ! $(command -v glide) ]; then
	echo "glide: command not found"
	exit 255
fi

if [ ! $(command -v glide-vc) ]; then
	echo "glide-vc: command not found"
	exit 255
fi

glide update --strip-vcs --strip-vendor --update-vendored --delete
glide-vc --only-code --no-tests --keep="**/*.json.in"
