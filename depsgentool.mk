$(call setup-stamp-file,DEPSGENTOOL_STAMP)

# variables for makelib/build_go_bin.mk
BGB_PKG_IN_REPO := tools/depsgen
BGB_BINARY := $(DEPSGENTOOL)

CLEAN_FILES += $(DEPSGENTOOL)

$(DEPSGENTOOL_STAMP): $(DEPSGENTOOL)
	touch "$@"

$(DEPSGENTOOL): $(MK_PATH) | $(TOOLSDIR)

include makelib/build_go_bin.mk

# DEPSGENTOOL_STAMP deliberately not cleared
