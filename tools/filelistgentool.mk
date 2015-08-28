$(call setup-stamp-file,FILELISTGENTOOL_STAMP)

# variables for makelib/build_go_bin.mk
BGB_PKG_IN_REPO := tools/filelistgen
BGB_BINARY := $(FILELISTGENTOOL)
BGB_ADDITIONAL_GO_ENV := GOARCH=$(GOARCH_FOR_BUILD)

CLEAN_FILES += $(FILELISTGENTOOL)

$(FILELISTGENTOOL_STAMP): $(FILELISTGENTOOL)
	touch "$@"

$(FILELISTGENTOOL): $(MK_PATH) | $(TOOLSDIR)

include makelib/build_go_bin.mk

# FILELISTGENTOOL_STAMP deliberately not cleared
