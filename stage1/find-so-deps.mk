$(call setup-stamp-file,STAGE1_FSD_STAMP)

STAGE1_FSD_LIBDIRS := $(ACIROOTFSDIR)/lib:$(ACIROOTFSDIR)/lib64:$(ACIROOTFSDIR)/usr/lib:$(ACIROOTFSDIR)/usr/lib64

ifneq ($(LD_LIBRARY_PATH),)

STAGE1_FSD_LIBDIRS := $(STAGE1_FSD_LIBDIRS):$(LD_LIBRARY_PATH)

endif

$(call forward-vars,$(STAGE1_FSD_STAMP), \
	ACIROOTFSDIR STAGE1_FSD_LIBDIRS INSTALL)
$(STAGE1_FSD_STAMP): $(STAGE1_STAMPS)
	set -e; \
	all_libs=$$(find "$(ACIROOTFSDIR)" -type f | xargs file | grep ELF | cut -f1 -d: | LD_LIBRARY_PATH="$(STAGE1_FSD_LIBDIRS)" xargs ldd | grep -v '^[^[:space:]]' | grep '/' | sed -e 's/^[[:space:]]*//' -e 's/[[:space:]]*(0x[0-9a-fA-F]*)//' -e 's/.*=>[[:space:]]*//' | grep -Fve "$(ACIROOTFSDIR)" | sort -u); \
	for f in $${all_libs}; do \
		$(INSTALL) -D "$${f}" "$(ACIROOTFSDIR)$${f}"; \
	done; \
	touch "$@"

# STAGE1_FSD_STAMP is deliberately not cleared - it will be used in
# stage1.mk to create stage1.aci's dependency on the stamp.
$(call undefine-namespaces,STAGE1_FSD _STAGE1_FSD,STAGE1_FSD_STAMP)
