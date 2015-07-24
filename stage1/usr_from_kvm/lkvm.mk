$(call setup-stamp-file,LKVM_STAMP)
LKVM_TMP := $(BUILDDIR)/tmp/usr_from_kvm/lkvm
LKVM_SRCDIR := $(LKVM_TMP)/src
LKVM_BINARY := $(LKVM_SRCDIR)/lkvm-static
LKVM_ACI_BINARY := $(ACIROOTFSDIR)/lkvm
LKVM_GIT := https://kernel.googlesource.com/pub/scm/linux/kernel/git/will/kvmtool
# just last published version (for reproducible builds), not for any other reason 
# cat be set to master to follow to upstream
# LKVM_VERSION := master
LKVM_VERSION := 4095fac878f618ae5e7384a1dc65ee34b6e05217

UFK_STAMPS += $(LKVM_STAMP)
INSTALL_FILES += $(LKVM_BINARY):$(LKVM_ACI_BINARY):-
CREATE_DIRS += $(LKVM_TMP)

$(LKVM_STAMP): $(LKVM_ACI_BINARY)
	touch "$@"

$(LKVM_BINARY): LKVM_SRCDIR := $(LKVM_SRCDIR)
$(LKVM_BINARY):
	$(MAKE) -C "$(LKVM_SRCDIR)" lkvm-static

$(LKVM_SRCDIR)/Makefile: LKVM_GIT := $(LKVM_GIT)
$(LKVM_SRCDIR)/Makefile: LKVM_SRCDIR := $(LKVM_SRCDIR)
$(LKVM_SRCDIR)/Makefile: LKVM_VERSION := $(LKVM_VERSION)
$(LKVM_SRCDIR)/Makefile: | $(LKVM_TMP)
	set -e; \
	mkdir -p $(LKVM_SRCDIR); cd $(LKVM_SRCDIR); \
	git init; \
	git remote add origin "$(LKVM_GIT)"; \
	git fetch --depth=1 origin $(LKVM_VERSION); \
	git checkout $(LKVM_VERSION)
	# git clone --depth=1 "$(LKVM_GIT)" "$(LKVM_SRCDIR)"

GR_TARGET := $(LKVM_BINARY)
GR_SRCDIR := $(LKVM_SRCDIR)
GR_BRANCH := master
GR_PREREQS := $(LKVM_SRCDIR)/Makefile

include makelib/git-refresh.mk

LKVM_STAMP :=
LKVM_TMP :=
LKVM_SRCDIR :=
LKVM_BINARY :=
LKVM_ACI_BINARY :=
LKVM_GIT :=
LKVM_VERSION := 
