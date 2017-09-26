#!/bin/bash
# Usage: [TOP_DIR=directory] sort-stage1-manifests.sh

set -e

: ${TOP_DIR:="$( cd "$( dirname "${BASH_SOURCE[0]}" )/.." && pwd )"}

tmp=$(mktemp --tmpdir manifest-sort-XXX)

cleanup () {
	rm -f ${tmp}
}

trap cleanup EXIT

boards='amd64-usr arm64-usr'
flavors='usr_from_coreos usr_from_kvm'

for board in ${boards}; do
	for flavor in ${flavors}; do
		dir="${TOP_DIR}/stage1/${flavor}/manifest-${board}.d/"
		files=$(find ${dir} -type f -name '*.manifest')
		for f in ${files}; do
			cat ${f} | sort | uniq | sed '/^$/d' > ${tmp}
			cp ${tmp} ${f}
		done
	done
done
