$(call setup-tmp-dir,LOCAL_TMPDIR)

LOCAL_FLAVOR := $(RKT_STAGE1_USR_FROM)
LOCAL_ETCDIR := $(ACIROOTFSDIR)/etc
LOCAL_ACI_OS_RELEASE := $(LOCAL_ETCDIR)/os-release
LOCAL_ACI_DIRS := \
	$(LOCAL_ETCDIR) \
	$(call dir-chain,$(ACIROOTFSDIR),opt/stage2) \
	$(call dir-chain,$(ACIROOTFSDIR),rkt/status) \
	$(call dir-chain,$(ACIROOTFSDIR),rkt/env)
LOCAL_ACI_MANIFEST := $(ACIDIR)/manifest
LOCAL_GEN_MANIFEST := $(LOCAL_TMPDIR)/aci-manifest
LOCAL_SRC_MANIFEST := $(MK_SRCDIR)/aci-manifest.in
LOCAL_NAME := coreos.com/rkt/stage1-$(LOCAL_FLAVOR)

ifeq ($(LOCAL_FLAVOR),src)

LOCAL_NAME := $(LOCAL_NAME)-$(RKT_STAGE1_SYSTEMD_VER)

endif

LOCAL_SED_NAME := $(call sed-replacement-escape,$(LOCAL_NAME))
LOCAL_SED_VERSION := $(call sed-replacement-escape,$(version))

$(call setup-stamp-file,LOCAL_STAMP)
$(call setup-stamp-file,LOCAL_MANIFEST_KV_DEPMK_STAMP,$(LOCAL_FLAVOR)-manifest-kv-dep)
$(call setup-dep-file,LOCAL_MANIFEST_KV_DEPMK,$(LOCAL_FLAVOR)-manifest-kv-dep)
# invalidate generated manifest if name, version or enter cmd changes
$(call generate-kv-deps,$(LOCAL_MANIFEST_KV_DEPMK_STAMP),$(LOCAL_GEN_MANIFEST),$(LOCAL_MANIFEST_KV_DEPMK),LOCAL_SED_NAME LOCAL_SED_VERSION)

$(LOCAL_STAMP): $(LOCAL_ACI_OS_RELEASE) $(LOCAL_ACI_MANIFEST) | $(LOCAL_ACI_DIRS)
	touch "$@"

$(call forward-vars,$(LOCAL_GEN_MANIFEST), \
	LOCAL_SED_NAME LOCAL_SED_VERSION)
$(LOCAL_GEN_MANIFEST): $(LOCAL_SRC_MANIFEST) | $(LOCAL_TMPDIR)
	set -e; \
	sed \
		-e 's/@RKT_STAGE1_NAME@/$(LOCAL_SED_NAME)/g' \
		-e 's/@RKT_STAGE1_VERSION@/$(LOCAL_SED_VERSION)/g' \
	"$<" >"$@.tmp"; \
	$(call bash-cond-rename,$@.tmp,$@)

STAGE1_INSTALL_DIRS += $(foreach d,$(LOCAL_ACI_DIRS),$d:0755)
STAGE1_INSTALL_FILES += \
	$(LOCAL_GEN_MANIFEST):$(LOCAL_ACI_MANIFEST):0644 \
	$(MK_SRCDIR)/os-release:$(LOCAL_ACI_OS_RELEASE):0644
STAGE1_STAMPS += $(LOCAL_STAMP)
CLEAN_FILES += $(LOCAL_GEN_MANIFEST)

$(call undefine-namespaces,LOCAL)
