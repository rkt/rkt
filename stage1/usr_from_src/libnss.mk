$(call setup-stamp-file,UFSN_STAMP)
UFSN_LIBNSS_FILES_LINK := libnss_files.so.2
# TODO: add this functionality to find-so-deps.mk
UFSN_LIBS := $(shell ld --verbose | grep SEARCH_DIR | sed -e 's/SEARCH_DIR("=*\([^"]*\)");*/\1/g')
UFSN_LIB_PATH := $(shell for l in $(UFSN_LIBS); do if [[ -e $${l}/$(UFSN_LIBNSS_FILES_LINK) ]]; then echo $${l}; break; fi; done)

UFSN_LIBNSS_FILES_PATH := $(UFSN_LIB_PATH)/$(UFSN_LIBNSS_FILES_LINK)
UFSN_LIBNSS_FILES_REALPATH := $(shell realpath $(UFSN_LIBNSS_FILES_PATH))
UFSN_LIBNSS_FILES := $(shell basename UFSN_LIBNSS_FILES_REALPATH)

UFSN_LIBNSS_FILES_ON_ACI := $(S1_RF_ACIROOTFSDIR)/usr/lib/$(UFSN_LIBNSS_FILES)

S1_RF_SECONDARY_STAMPS += $(UFSN_STAMP)
S1_RF_INSTALL_FILES += $(UFSN_LIBNSS_FILES_PATH):$(UFSN_LIBNSS_FILES_ON_ACI):-
S1_RF_INSTALL_SYMLINKS += $(UFSN_LIBNSS_FILES):$(S1_RF_ACIROOTFSDIR)/usr/lib/$(UFSN_LIBNSS_FILES_LINK)

$(call generate-stamp-rule,$(UFSN_STAMP),$(UFSN_LIBNSS_FILES_ON_ACI), $(S1_RF_ACIROOTFSDIR)/usr/lib/$(UFSN_LIBNSS_FILES_LINK))

$(call undefine-namespaces,UFSN)
