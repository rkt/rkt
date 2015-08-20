# inputs cleared after including this file:
# BGB_BINARY
# BGB_GO_FLAGS
# BGB_PKG_IN_REPO
# BGB_ADDITIONAL_GO_ENV

# inputs left alone:
# DEPSDIR
# DEPSGENTOOL
# GOPATH
# GO_ENV
# MAKEFILE_LIST
# REPO_PATH

_BGB_PATH_ := $(lastword $(MAKEFILE_LIST))

# the gopath symlink creation rule should be generated only once, even
# if we include this file multiple times.
ifeq ($(_BGB_RKT_SYMLINK_STAMP_),)

# the symlink stamp wasn't yet generated, do it now.

_BGB_RKT_SYMLINK_NAME_ := $(GOPATH)/src/$(REPO_PATH)

$(call setup-custom-stamp-file,_BGB_RKT_SYMLINK_STAMP_,$(_BGB_PATH_)/rkt-symlink)

$(_BGB_RKT_SYMLINK_STAMP_): | $(_BGB_RKT_SYMLINK_NAME_)
	touch "$@"

INSTALL_SYMLINKS += $(MK_TOPLEVEL_ABS_SRCDIR):$(_BGB_RKT_SYMLINK_NAME_)
CREATE_DIRS += $(call to-dir,$(_BGB_RKT_SYMLINK_NAME_))

endif

_BGB_PKG_NAME_ := $(REPO_PATH)/$(BGB_PKG_IN_REPO)

$(call setup-dep-file,_BGB_DEPMK,$(_BGB_PKG_NAME_))
$(call setup-dep-file,_BGB_KV_DEPMK,$(_BGB_PKG_NAME_)/kv)

# Do not depend on depsgen when we are building depsgen. Also, when
# building depsgen, it will be built first as depsgen.tmp, which in
# turn will be run to get the dependencies and then it will be renamed
# to depsgen.
ifeq ($(BGB_BINARY),$(DEPSGENTOOL))

_BGB_DEPSGEN_SUFFIX_ := .tmp

else

$(BGB_BINARY): $(DEPSGENTOOL_STAMP)

endif

-include $(_BGB_DEPMK)
-include $(_BGB_KV_DEPMK)
$(call forward-vars,$(BGB_BINARY), \
	BGB_ADDITIONAL_GO_ENV GO_ENV GO BGB_GO_FLAGS _BGB_PKG_NAME_ DEPSGENTOOL \
	_BGB_DEPSGEN_SUFFIX_ REPO_PATH BGB_PKG_IN_REPO _BGB_DEPMK _BGB_KV_DEPMK)
$(BGB_BINARY): _ESCAPED_BGB_GO_FLAGS_ := $(call escape-and-wrap,$(BGB_GO_FLAGS))
$(BGB_BINARY): $(_BGB_PATH_) $(_BGB_RKT_SYMLINK_STAMP_) | $(DEPSDIR)
	set -e; \
	$(BGB_ADDITIONAL_GO_ENV) $(GO_ENV) "$(GO)" build -o "$@.tmp" $(BGB_GO_FLAGS) "$(_BGB_PKG_NAME_)"; \
	$(GO_ENV) "$(DEPSGENTOOL)$(_BGB_DEPSGEN_SUFFIX_)" go --repo "$(REPO_PATH)" --module "$(BGB_PKG_IN_REPO)" --target '$$(BGB_BINARY)' >"$(_BGB_DEPMK)"; \
	"$(DEPSGENTOOL)$(_BGB_DEPSGEN_SUFFIX_)" kv --target '$$(BGB_BINARY)' BGB_GO_FLAGS $(_ESCAPED_BGB_GO_FLAGS_) >"$(_BGB_KV_DEPMK)"; \
	mv "$@.tmp" "$@"

# _BGB_RKT_SYMLINK_STAMP_ is deliberately not cleared - it needs to
# stay defined to make sure that the gopath symlink rule is generated
# only once.
$(call undefine-namespaces,BGB _BGB,_BGB_RKT_SYMLINK_STAMP_)
