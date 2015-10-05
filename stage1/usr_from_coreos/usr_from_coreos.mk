$(call inc-one,coreos-common.mk)

$(call setup-tmp-dir,UFC_TMPDIR)

UFC_ROOTFS := $(UFC_TMPDIR)/rootfs
UFC_FILELIST := $(UFC_TMPDIR)/manifest.txt
UFC_MANIFESTS_DIR ?= $(MK_SRCDIR)/manifest.d
UFC_MANIFESTS := $(wildcard $(UFC_MANIFESTS_DIR)/*.manifest)

$(call setup-dep-file,UFC_DEPMK,manifests)
$(call setup-clean-file,UFC_ROOTFSDIR_CLEANMK,/rootfs)
$(call setup-clean-file,UFC_ACIROOTFSDIR_CLEANMK,/acirootfs)
$(call setup-stamp-file,UFC_STAMP)
$(call setup-stamp-file,UFC_MKBASE_STAMP,/mkbase)
$(call setup-stamp-file,UFC_ACI_ROOTFS_STAMP,/acirootfs)
$(call setup-stamp-file,UFC_ACIROOTFS_DEPS_STAMP,/acirootfs-deps)
$(call setup-stamp-file,UFC_ROOTFS_CLEAN_STAMP,/rootfs-clean)
$(call setup-stamp-file,UFC_ACIROOTFS_CLEAN_STAMP,/acirootfs-clean)

$(call setup-filelist-file,UFC_DETAILED_FILELIST)

INSTALL_DIRS += $(UFC_ITMP):-
S1_RF_USR_STAMPS += $(UFC_STAMP)
CLEAN_FILES += \
	$(UFC_FILELIST) \
	$(S1_RF_ACIROOTFSDIR)/systemd-version
CLEAN_SYMLINKS += \
	$(S1_RF_ACIROOTFSDIR)/flavor \
	$(S1_RF_ACIROOTFSDIR)/lib64 \
	$(S1_RF_ACIROOTFSDIR)/lib \
	$(S1_RF_ACIROOTFSDIR)/bin
CLEAN_DIRS += \
	$(UFC_ROOTFS)

$(UFC_STAMP): $(UFC_ACI_ROOTFS_STAMP) $(UFC_ACIROOTFS_DEPS_STAMP) $(UFC_ACIROOTFS_CLEAN_STAMP) $(UFC_ROOTFS_CLEAN_STAMP)
	touch "$@"

$(call forward-vars,$(UFC_ACI_ROOTFS_STAMP), \
	S1_RF_ACIROOTFSDIR UFC_ROOTFS CCN_SYSTEMD_VERSION)
$(UFC_ACI_ROOTFS_STAMP): $(UFC_MKBASE_STAMP) $(UFC_FILELIST)
	set -e; \
	rm -rf "$(S1_RF_ACIROOTFSDIR)"; \
	cp -af "$(UFC_ROOTFS)/." "$(S1_RF_ACIROOTFSDIR)"; \
	 \
	ln -sf 'coreos' "$(S1_RF_ACIROOTFSDIR)/flavor"; \
	ln -sf 'usr/lib64' "$(S1_RF_ACIROOTFSDIR)/lib64"; \
	ln -sf 'usr/lib' "$(S1_RF_ACIROOTFSDIR)/lib"; \
	ln -sf 'usr/bin' "$(S1_RF_ACIROOTFSDIR)/bin"; \
	echo "$(CCN_SYSTEMD_VERSION)" >"$(S1_RF_ACIROOTFSDIR)/systemd-version"; \
	touch "$@"

# This depmk can be created only when detailed filelist is generated
$(UFC_ACIROOTFS_DEPS_STAMP): $(UFC_DETAILED_FILELIST)
$(call generate-glob-deps,$(UFC_ACIROOTFS_DEPS_STAMP),$(UFC_ACI_ROOTFS_STAMP),$(UFC_DEPMK),,$(UFC_DETAILED_FILELIST),$(UFC_ROOTFS))

# This cleanmk can be created only when detailed filelist is generated
$(UFC_ACIROOTFS_CLEAN_STAMP): $(UFC_DETAILED_FILELIST)
$(call generate-clean-mk,$(UFC_ACIROOTFS_CLEAN_STAMP),$(UFC_ACIROOTFSDIR_CLEANMK),$(UFC_DETAILED_FILELIST),$(S1_RF_ACIROOTFSDIR))

$(call forward-vars,$(UFC_MKBASE_STAMP), \
	UFC_ROOTFS UFC_FILELIST CCN_SQUASHFS)
$(UFC_MKBASE_STAMP): $(CCN_SQUASHFS) $(UFC_FILELIST)
	set -e; \
	rm -rf "$(UFC_ROOTFS)"; \
	install -m 0750 -d "$(UFC_ROOTFS)"; \
	unsquashfs -d "$(UFC_ROOTFS)/usr" -ef "$(UFC_FILELIST)" "$(CCN_SQUASHFS)"; \
	touch "$@"

# This filelist can be generated only after the pxe image was
# unpackaged and unsquashed
$(UFC_DETAILED_FILELIST): $(UFC_MKBASE_STAMP)
$(call generate-deep-filelist,$(UFC_DETAILED_FILELIST),$(UFC_ROOTFS))

# This cleanmk can be generated only after the detailed filelist was
# generated.
$(UFC_ROOTFS_CLEAN_STAMP): $(UFC_DETAILED_FILELIST)
$(call generate-clean-mk,$(UFC_ROOTFS_CLEAN_STAMP),$(UFC_ROOTFSDIR_CLEANMK),$(UFC_DETAILED_FILELIST),$(UFC_ROOTFS))

$(UFC_FILELIST): $(UFC_MANIFESTS) | $(UFC_TMPDIR)
	cat $^ | sort -u > "$@.tmp"
	cmp "$@.tmp" "$@" || mv "$@.tmp" "$@"
	rm -f "$@.tmp"

$(call undefine-namespaces,UFC)
