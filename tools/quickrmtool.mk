$(call setup-stamp-file,QUICKRMTOOL_STAMP)

# variables for makelib/build_go_bin.mk
BGB_STAMP := $(QUICKRMTOOL_STAMP)
BGB_PKG_IN_REPO := tools/quickrm
BGB_BINARY := $(QUICKRMTOOL)
BGB_ADDITIONAL_GO_ENV := GOARCH=$(GOARCH_FOR_BUILD)

CLEAN_FILES += $(QUICKRMTOOL)

$(QUICKRMTOOL_STAMP):
	touch "$@"

$(QUICKRMTOOL): $(MK_PATH) | $(TOOLSDIR)

include makelib/build_go_bin.mk

# QUICKRMTOOL_STAMP deliberately not cleared
