UFK_STAMPS :=
UFK_INCLUDES := \
	../usr_from_coreos/usr_from_coreos.mk \
	kernel.mk \
	lkvm.mk
$(call setup-stamp-file,UFK_REPLACE_FLAVOR_STAMP)

$(call inc-many,$(UFK_INCLUDES))

STAGE1_STAMPS += $(UFK_REPLACE_FLAVOR_STAMP)
STAGE1_USR_STAMPS += $(UFK_STAMPS)

$(call forward-vars,$(UFK_REPLACE_FLAVOR_STAMP), \
	ACIROOTFSDIR)
$(UFK_REPLACE_FLAVOR_STAMP):
	rm -f "$(ACIROOTFSDIR)/flavor"
	ln -sf 'kvm' "$(ACIROOTFSDIR)/flavor"
	touch "$@"

$(call undefine-namespaces,UFK)
