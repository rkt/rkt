LOCAL_ETCDIR := $(ACIROOTFSDIR)/etc
LOCAL_ACI_OS_RELEASE := $(LOCAL_ETCDIR)/os-release
LOCAL_ACI_DIRS := \
	$(LOCAL_ETCDIR) \
	$(call dir-chain,$(ACIROOTFSDIR),opt/stage2) \
	$(call dir-chain,$(ACIROOTFSDIR),rkt/status) \
	$(call dir-chain,$(ACIROOTFSDIR),rkt/env)
LOCAL_ACI_MANIFEST := $(ACIDIR)/manifest

$(call setup-stamp-file,LOCAL_STAMP)

$(LOCAL_STAMP): $(LOCAL_ACI_OS_RELEASE) $(LOCAL_ACI_MANIFEST) | $(LOCAL_ACI_DIRS)
	touch "$@"

STAGE1_INSTALL_DIRS += $(foreach d,$(LOCAL_ACI_DIRS),$d:0755)
STAGE1_INSTALL_FILES += \
	$(MK_SRCDIR)/aci-manifest:$(LOCAL_ACI_MANIFEST):0644 \
	$(MK_SRCDIR)/os-release:$(LOCAL_ACI_OS_RELEASE):0644
STAGE1_STAMPS += $(LOCAL_STAMP)

$(call undefine-namespaces,LOCAL)
