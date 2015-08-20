$(call setup-stamp-file,LKVM_STAMP)
LKVM_TMP := $(BUILDDIR)/tmp/usr_from_kvm/lkvm
LKVM_SRCDIR := $(LKVM_TMP)/src
LKVM_BINARY := $(LKVM_SRCDIR)/lkvm-static
LKVM_ACI_BINARY := $(ACIROOTFSDIR)/lkvm
LKVM_GIT := https://kernel.googlesource.com/pub/scm/linux/kernel/git/will/kvmtool
# just last published version (for reproducible builds), not for any other reason
LKVM_VERSION := 4095fac878f618ae5e7384a1dc65ee34b6e05217

LKVM_STUFFDIR := $(MK_SRCDIR)/lkvm
LKVM_PATCHES := $(abspath $(LKVM_STUFFDIR)/patches/*.patch)

$(call setup-stamp-file,LKVM_PATCH_STAMP,/patch_lkvm)
$(call setup-dep-file,LKVM_PATCHES_DEPMK)

UFK_STAMPS += $(LKVM_STAMP)
INSTALL_FILES += $(LKVM_BINARY):$(LKVM_ACI_BINARY):-
CREATE_DIRS += $(LKVM_TMP)


$(LKVM_STAMP): $(LKVM_ACI_BINARY)
	touch "$@"

$(call forward-vars,$(LKVM_BINARY), \
	MAKE LKVM_SRCDIR)
$(LKVM_BINARY): $(LKVM_PATCH_STAMP)
	$(MAKE) -C "$(LKVM_SRCDIR)" lkvm-static

-include $(LKVM_PATCHES_DEPMK)
$(call forward-vars,$(LKVM_PATCH_STAMP), \
	DEPSGENTOOL LKVM_PATCHES LKVM_PATCHES_DEPMK LKVM_SRCDIR)
$(LKVM_PATCH_STAMP): $(LKVM_SRCDIR)/Makefile $(DEPSGENTOOL_STAMP)
	set -e; \
	shopt -s nullglob; \
	"$(DEPSGENTOOL)" glob --target '$$(LKVM_SRCDIR)/Makefile' --suffix=.patch $(LKVM_PATCHES) >"$(LKVM_PATCHES_DEPMK)"; \
	for p in $(LKVM_PATCHES); do \
		patch --directory="$(LKVM_SRCDIR)" --strip=1 --forward <"$${p}"; \
	done; \
	touch "$@"

# add remote only if not added
# don't fetch existing (commit cannot change)
$(call forward-vars,$(LKVM_SRCDIR)/Makefile, \
	LKVM_SRCDIR LKVM_GIT LKVM_VERSION)
$(LKVM_SRCDIR)/Makefile: | $(LKVM_TMP)
	set -e; \
	mkdir -p $(LKVM_SRCDIR); cd $(LKVM_SRCDIR); \
	git init; \
	git remote | grep --silent origin || git remote add origin "$(LKVM_GIT)"; \
	git rev-parse --quiet --verify HEAD >/dev/null || git fetch --depth=1 origin $(LKVM_VERSION) && git checkout --quiet $(LKVM_VERSION); \
	git reset --hard; \
	git clean -ffdx; \
	touch "$@"

$(call undefine-namespaces,LKVM)
