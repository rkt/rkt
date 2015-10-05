$(call setup-stamp-file,UFH_STAMP)

S1_RF_USR_STAMPS += $(UFH_STAMP)

$(call forward-vars,$(UFH_STAMP), \
	S1_RF_ACIROOTFSDIR)
$(UFH_STAMP): | $(S1_RF_ACIROOTFSDIR)
	ln -sf 'host' "$(S1_RF_ACIROOTFSDIR)/flavor"
	touch "$@"

CLEAN_SYMLINKS += $(S1_RF_ACIROOTFSDIR)/flavor

$(call undefine-namespaces,UFH)
