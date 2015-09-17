$(call setup-clean-file,STAGE1_FSD_SOLIBS_CLEANMK,/solibs)

$(call setup-tmp-dir,STAGE1_FSD_TMPDIR)

# This is where libraries from host are copied too, so we can prepare
# an exact filelist of what exactly was copied.
STAGE1_FSD_LIBSDIR := $(STAGE1_FSD_TMPDIR)/libs

$(call setup-stamp-file,STAGE1_FSD_STAMP)
$(call setup-stamp-file,STAGE1_FSD_COPY_STAMP,/fsd_copy)
$(call setup-stamp-file,STAGE1_FSD_CLEAN_STAMP,/fsd_clean)

$(call setup-filelist-file,STAGE1_FSD_FILELIST,/fsd)

STAGE1_FSD_LD_LIBRARY_PATH := $(ACIROOTFSDIR)/lib:$(ACIROOTFSDIR)/lib64:$(ACIROOTFSDIR)/usr/lib:$(ACIROOTFSDIR)/usr/lib64

ifneq ($(LD_LIBRARY_PATH),)

STAGE1_FSD_LD_LIBRARY_PATH := $(STAGE1_FSD_LD_LIBRARY_PATH):$(LD_LIBRARY_PATH)

endif

CREATE_DIRS += $(STAGE1_FSD_LIBSDIR)

$(STAGE1_FSD_STAMP): $(STAGE1_FSD_COPY_STAMP) $(STAGE1_FSD_CLEAN_STAMP)
	touch "$@"

$(call forward-vars,$(STAGE1_FSD_COPY_STAMP), \
	ACIROOTFSDIR STAGE1_FSD_LD_LIBRARY_PATH INSTALL STAGE1_FSD_LIBSDIR)
$(STAGE1_FSD_COPY_STAMP): $(STAGE1_STAMPS) | $(STAGE1_FSD_LIBSDIR)
	set -e; \
	all_libs=$$(find "$(ACIROOTFSDIR)" -type f | xargs file | grep ELF | cut -f1 -d: | LD_LIBRARY_PATH="$(STAGE1_FSD_LD_LIBRARY_PATH)" xargs ldd | grep -v '^[^[:space:]]' | grep '/' | sed -e 's/^[[:space:]]*//' -e 's/[[:space:]]*(0x[0-9a-fA-F]*)//' -e 's/.*=>[[:space:]]*//' | grep -Fve "$(ACIROOTFSDIR)" | sort -u); \
	for f in $${all_libs}; do \
		$(INSTALL) -D "$${f}" "$(ACIROOTFSDIR)$${f}"; \
		$(INSTALL) -D "$${f}" "$(STAGE1_FSD_LIBSDIR)$${f}"; \
	done; \
	touch "$@"

# This filelist can be generated only after the files were copied.
$(STAGE1_FSD_FILELIST): $(STAGE1_FSD_COPY_STAMP)
$(call generate-deep-filelist,$(STAGE1_FSD_FILELIST),$(STAGE1_FSD_LIBSDIR))

# Generate clean.mk file cleaning libraries copied from host to both
# temporary directory and ACI rootfs directory.
$(call generate-clean-mk,$(STAGE1_FSD_CLEAN_STAMP),$(STAGE1_FSD_SOLIBS_CLEANMK),$(STAGE1_FSD_FILELIST),$(STAGE1_FSD_LIBSDIR) $(ACIROOTFSDIR))

# STAGE1_FSD_STAMP is deliberately not cleared - it will be used in
# stage1.mk to create stage1.aci's dependency on the stamp.
$(call undefine-namespaces,STAGE1_FSD _STAGE1_FSD,STAGE1_FSD_STAMP)
