$(call setup-stamp-file,UFH_STAMP)

STAGE1_USR_STAMPS += $(UFH_STAMP)

$(call forward-vars,$(UFH_STAMP), \
	ACIROOTFSDIR)
$(UFH_STAMP): | $(ACIROOTFSDIR)
	ln -sf 'host' "$(ACIROOTFSDIR)/flavor"
	touch "$@"

CLEAN_SYMLINKS += $(ACIROOTFSDIR)/flavor

$(call undefine-namespaces,UFH)
