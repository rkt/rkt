# custom kernel compilation
KERNEL_VERSION := 4.1.3
KERNEL_TMPDIR := $(UFK_TMPDIR)/kernel
KERNEL_NAME := linux-$(KERNEL_VERSION)
KERNEL_TARBALL := $(KERNEL_NAME).tar.xz
KERNEL_TARGET_FILE := $(KERNEL_TMPDIR)/$(KERNEL_TARBALL)
KERNEL_SRCDIR := $(KERNEL_TMPDIR)/$(KERNEL_NAME)
KERNEL_BUILDDIR := $(abspath $(KERNEL_TMPDIR)/build-$(KERNEL_VERSION))
KERNEL_URL := https://www.kernel.org/pub/linux/kernel/v4.x/$(KERNEL_TARBALL)
KERNEL_MAKEFILE := $(KERNEL_SRCDIR)/Makefile
KERNEL_STUFFDIR := $(MK_SRCDIR)/kernel
KERNEL_SRC_CONFIG := $(KERNEL_STUFFDIR)/cutdown-config
KERNEL_PATCHESDIR := $(KERNEL_STUFFDIR)/patches
KERNEL_PATCHES := $(abspath $(KERNEL_PATCHESDIR)/*.patch)
KERNEL_BUILD_CONFIG := $(KERNEL_BUILDDIR)/.config
KERNEL_BZIMAGE := $(KERNEL_BUILDDIR)/arch/x86/boot/bzImage
KERNEL_ACI_BZIMAGE := $(ACIROOTFSDIR)/bzImage

$(call setup-stamp-file,KERNEL_STAMP,/build_kernel)
$(call setup-stamp-file,KERNEL_BZIMAGE_STAMP,/bzimage)
$(call setup-stamp-file,KERNEL_PATCH_STAMP,/patch_kernel)
$(call setup-stamp-file,KERNEL_DEPS_STAMP,/deps)
$(call setup-dep-file,KERNEL_PATCHES_DEPMK)
$(call setup-clean-file,KERNEL_SRC_CLEANMK,/src)
$(call setup-clean-file,KERNEL_BUILD_CLEANMK,/build)
$(call setup-filelist-file,KERNEL_PATCHES_FILELIST,/patches)
$(call setup-filelist-file,KERNEL_BUILD_FILELIST,/build)
$(call setup-filelist-file,KERNEL_SRC_FILELIST,/src)
$(call setup-stamp-file,KERNEL_SRC_CLEAN_STAMP,/src-clean)
$(call setup-stamp-file,KERNEL_BUILD_CLEAN_STAMP,/build-clean)

CREATE_DIRS += $(KERNEL_TMPDIR) $(KERNEL_BUILDDIR)
CLEAN_DIRS += $(KERNEL_SRCDIR)
INSTALL_FILES += \
	$(KERNEL_SRC_CONFIG):$(KERNEL_BUILD_CONFIG):- \
	$(KERNEL_BZIMAGE):$(KERNEL_ACI_BZIMAGE):-
UFK_STAMPS += $(KERNEL_STAMP)
CLEAN_FILES += $(KERNEL_TARGET_FILE)

$(KERNEL_STAMP): $(KERNEL_ACI_BZIMAGE) $(KERNEL_DEPS_STAMP) $(KERNEL_BUILD_CLEAN_STAMP) $(KERNEL_SRC_CLEAN_STAMP)
	touch "$@"

# $(KERNEL_ACI_BZIMAGE) has a dependency on $(KERNEL_BZIMAGE), which
# is actually provided by $(KERNEL_BZIMAGE_STAMP)
$(KERNEL_BZIMAGE): $(KERNEL_BZIMAGE_STAMP)

# This stamp is to make sure that building linux kernel has finished.
$(call forward-vars,$(KERNEL_BZIMAGE), \
	MAKE KERNEL_SRCDIR KERNEL_BUILDDIR)
$(KERNEL_BZIMAGE_STAMP): $(KERNEL_BUILD_CONFIG) $(KERNEL_PATCH_STAMP)
	set -e; \
	$(MAKE) -C "$(KERNEL_SRCDIR)" O="$(KERNEL_BUILDDIR)" bzImage; \
	touch "$@"

# Generate filelist of a builddir. Can happen only after the building
# finished.
$(KERNEL_BUILD_FILELIST): $(KERNEL_BZIMAGE_STAMP)
$(call generate-deep-filelist,$(KERNEL_BUILD_FILELIST),$(KERNEL_BUILDDIR))

# Generate clean.mk cleaning builddir.
$(call generate-clean-mk,$(KERNEL_BUILD_CLEAN_STAMP),$(KERNEL_BUILD_CLEANMK),$(KERNEL_BUILD_FILELIST),$(KERNEL_BUILDDIR))

$(call forward-vars,$(KERNEL_PATCH_STAMP), \
	KERNEL_PATCHES KERNEL_SRCDIR)
$(KERNEL_PATCH_STAMP): $(KERNEL_MAKEFILE)
	set -e; \
	shopt -s nullglob; \
	for p in $(KERNEL_PATCHES); do \
		patch --directory="$(KERNEL_SRCDIR)" --strip=1 --forward <"$${p}"; \
	done; \
	touch "$@"

# Generate a filelist of srcdir. Can happen after the sources were
# patched.
$(KERNEL_SRC_FILELIST): $(KERNEL_PATCH_STAMP)
$(call generate-deep-filelist,$(KERNEL_SRC_FILELIST),$(KERNEL_SRCDIR))

# Generate clean.mk cleaning the sources.
$(call generate-clean-mk,$(KERNEL_SRC_CLEAN_STAMP),$(KERNEL_SRC_CLEANMK),$(KERNEL_SRC_FILELIST),$(KERNEL_SRCDIR))

# This is a special case - normally, when generating filelists, we
# require the directory to exist. In this case, the patches directory
# may not exist and it is fine. We generate an empty filelist.
KERNEL_GOT_PATCHES := $(shell test -d "$(KERNEL_PATCHESDIR)" && echo yes)

ifeq ($(KERNEL_GOT_PATCHES),yes)

# Generate a filelist of patches. Can happen anytime.
$(call generate-shallow-filelist,$(KERNEL_PATCHES_FILELIST),$(KERNEL_PATCHESDIR),.patch)

else

# Generate empty filelist of patches. This can happen anytime.
$(call generate-empty-filelist,$(KERNEL_PATCHES_FILELIST))

endif

# Generate a dep.mk on those patches, so if patches change, sources
# are removed, extracted again and repatched.
$(call generate-glob-deps,$(KERNEL_DEPS_STAMP),$(KERNEL_MAKEFILE),$(KERNEL_PATCHES_DEPMK),.patch,$(KERNEL_PATCHES_FILELIST),$(KERNEL_PATCHESDIR),normal)

$(call forward-vars,$(KERNEL_MAKEFILE), \
	KERNEL_SRCDIR KERNEL_TMPDIR)
$(KERNEL_MAKEFILE): $(KERNEL_TARGET_FILE)
	set -e; \
	rm -rf "$(KERNEL_SRCDIR)"; \
	tar --extract --xz --touch --file="$<" --directory="$(KERNEL_TMPDIR)"

$(call forward-vars,$(KERNEL_TARGET_FILE), \
	KERNEL_URL)
$(KERNEL_TARGET_FILE): | $(KERNEL_TMPDIR)
	wget --tries=20 --output-document="$@" "$(KERNEL_URL)"

$(call undefine-namespaces,KERNEL)
