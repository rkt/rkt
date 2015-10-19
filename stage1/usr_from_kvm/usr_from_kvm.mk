UFK_INCLUDES := \
	../usr_from_coreos/usr_from_coreos.mk \
	kernel.mk \
	files.mk \
	lkvm.mk
$(call setup-stamp-file,UFK_REPLACE_FLAVOR_STAMP)
$(call setup-tmp-dir,UFK_TMPDIR)

UFC_MANIFESTS_DIR := $(MK_SRCDIR)/manifest.d
$(call inc-many,$(UFK_INCLUDES))

S1_RF_SECONDARY_STAMPS += $(UFK_REPLACE_FLAVOR_STAMP)

$(call forward-vars,$(UFK_REPLACE_FLAVOR_STAMP), \
	S1_RF_ACIROOTFSDIR)
$(UFK_REPLACE_FLAVOR_STAMP):
	rm -f "$(S1_RF_ACIROOTFSDIR)/flavor"
	ln -sf 'kvm' "$(S1_RF_ACIROOTFSDIR)/flavor"
	touch "$@"

$(call undefine-namespaces,UFK)
