$(call setup-stamp-file,LKVM_STAMP)
LKVM_TMPDIR := $(UFK_TMPDIR)/lkvm
LKVM_SRCDIR := $(LKVM_TMPDIR)/src
LKVM_BINARY := $(LKVM_SRCDIR)/lkvm-static
LKVM_ACI_BINARY := $(ACIROOTFSDIR)/lkvm
LKVM_GIT := https://kernel.googlesource.com/pub/scm/linux/kernel/git/will/kvmtool
# just last published version (for reproducible builds), not for any other reason
LKVM_VERSION := efcf862611f2498d7b500e46a73d8a008e04325f

LKVM_STUFFDIR := $(MK_SRCDIR)/lkvm
LKVM_PATCHESDIR := $(LKVM_STUFFDIR)/patches
LKVM_PATCHES := $(abspath $(LKVM_PATCHESDIR)/*.patch)

$(call setup-stamp-file,LKVM_BUILD_STAMP,/build)
$(call setup-stamp-file,LKVM_PATCH_STAMP,/patch_lkvm)
$(call setup-stamp-file,LKVM_DEPS_STAMP,/deps)
$(call setup-stamp-file,LKVM_DIR_CLEAN_STAMP,/dir-clean)
$(call setup-dep-file,LKVM_PATCHES_DEPMK)
$(call setup-clean-file,LKVM_CLEANMK,/src)
$(call setup-filelist-file,LKVM_DIR_FILELIST,/dir)
$(call setup-filelist-file,LKVM_PATCHES_FILELIST,/patches)

UFK_STAMPS += $(LKVM_STAMP)
INSTALL_FILES += $(LKVM_BINARY):$(LKVM_ACI_BINARY):-
CREATE_DIRS += $(LKVM_TMPDIR)
CLEAN_DIRS += $(LKVM_SRCDIR)

$(LKVM_STAMP): $(LKVM_ACI_BINARY) $(LKVM_DEPS_STAMP) $(LKVM_DIR_CLEAN_STAMP)
	touch "$@"

$(LKVM_BINARY): $(LKVM_BUILD_STAMP)

$(call forward-vars,$(LKVM_BUILD_STAMP), \
	MAKE LKVM_SRCDIR)
$(LKVM_BUILD_STAMP): $(LKVM_PATCH_STAMP)
	set -e; \
	$(MAKE) -C "$(LKVM_SRCDIR)" lkvm-static; \
	touch "$@"

# Generate filelist of lkvm directory (this is both srcdir and
# builddir). Can happen after build finished.
$(LKVM_DIR_FILELIST): $(LKVM_BUILD_STAMP)
$(call generate-deep-filelist,$(LKVM_DIR_FILELIST),$(LKVM_SRCDIR))

# Generate clean.mk cleaning lkvm directory
$(call generate-clean-mk,$(LKVM_DIR_CLEAN_STAMP),$(LKVM_CLEANMK),$(LKVM_DIR_FILELIST),$(LKVM_SRCDIR))

$(call forward-vars,$(LKVM_PATCH_STAMP), \
	LKVM_PATCHES LKVM_SRCDIR)
$(LKVM_PATCH_STAMP): $(LKVM_SRCDIR)/Makefile
	set -e; \
	shopt -s nullglob; \
	for p in $(LKVM_PATCHES); do \
		patch --directory="$(LKVM_SRCDIR)" --strip=1 --forward <"$${p}"; \
	done; \
	touch "$@"

# This is a special case - normally, when generating filelists, we
# require the directory to exist. In this case, the patches directory
# may not exist and it is fine. We generate an empty filelist.
LKVM_GOT_PATCHES := $(shell test -d "$(LKVM_PATCHESDIR)" && echo yes)

ifeq ($(KERNEL_GOT_PATCHES),yes)

# Generate shallow filelist of patches. Can happen anytime.
$(call generate-shallow-filelist,$(LKVM_PATCHES_FILELIST),$(LKVM_PATCHESDIR),.patch)

else

# Generate empty filelist of patches. This can happen anytime.
$(call generate-empty-filelist,$(LKVM_PATCHES_FILELIST))

endif

# Generate dep.mk on patches, so if they change, the project has to be
# reset to original checkout and patches reapplied.
$(call generate-glob-deps,$(LKVM_DEPS_STAMP),$(LKVM_SRCDIR)/Makefile,$(LKVM_PATCHES_DEPMK),.patch,$(LKVM_PATCHES_FILELIST),$(LKVM_PATCHESDIR),normal)

# add remote only if not added
# don't fetch existing (commit cannot change)
$(call forward-vars,$(LKVM_SRCDIR)/Makefile, \
	LKVM_SRCDIR LKVM_GIT LKVM_VERSION)
$(LKVM_SRCDIR)/Makefile: | $(LKVM_TMPDIR)
	set -e; \
	mkdir -p $(LKVM_SRCDIR); cd $(LKVM_SRCDIR); \
	git init; \
	git remote | grep --silent origin || git remote add origin "$(LKVM_GIT)"; \
	git rev-parse --quiet --verify HEAD >/dev/null || git fetch --depth=1 origin $(LKVM_VERSION) && git checkout --quiet $(LKVM_VERSION); \
	git reset --hard; \
	git clean -ffdx; \
	touch "$@"

$(call undefine-namespaces,LKVM)
