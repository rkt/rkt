# Creates a rule creating directories.
# Example: $(eval $(call _FILE_OPS_CREATE_DIRS_RULE_,dir1 dir1/dir2 dir1/dir2/dir3))
define _FILE_OPS_CREATE_DIRS_RULE_
CLEAN_DIRS += $1
$1:
	[[ -e "$$@" ]] || mkdir -p "$$@"
	$$(call _FILE_OPS_BAIL_OUT_IF_NOT_DIR_,$$@)
endef

# Creates a rule for installing directory. Depends on parent
# directory.
# Example: $(eval $(call _FILE_OPS_CREATE_INSTALL_DIR_RULE_,dir:0644))
define _FILE_OPS_CREATE_INSTALL_DIR_RULE_
$$(eval $$(call _FILE_OPS_SPLIT_2_,$1,_FILE_OPS_CIDR_DIR_,_FILE_OPS_CIDR_MODE_))
CLEAN_DIRS += $$(_FILE_OPS_CIDR_DIR_)
$$(call forward-vars,$$(_FILE_OPS_CIDR_DIR_), \
	INSTALL _FILE_OPS_CIDR_MODE_)
# TODO: Create a proper dependency on parent directory
# $$(_FILE_OPS_CIDR_DIR_): | $$(call to-dir,$$(_FILE_OPS_CIDR_DIR_))
$$(_FILE_OPS_CIDR_DIR_):
	[[ -e "$$@" ]] || $$(INSTALL) $$(call _FILE_OPS_DASH_M_,$$(_FILE_OPS_CIDR_MODE_)) -d "$$@"
	$$(call _FILE_OPS_BAIL_OUT_IF_NOT_DIR_,$$@)
$$(call undefine-namespaces,_FILE_OPS_CIDR)
endef

# Creates a rule for installing a file. Depends on source file and
# parent directory. Pass - as a third parameter for dest to inherit
# mode from src.
# Example: $(eval $(call _FILE_OPS_CREATE_INSTALL_FILE_RULE_,src,dest,0755))
define _FILE_OPS_CREATE_INSTALL_FILE_RULE_
$$(eval $$(call _FILE_OPS_SPLIT_3_,$1,_FILE_OPS_CIFR_SRC_,_FILE_OPS_CIFR_DEST_,_FILE_OPS_CIFR_MODE_))
CLEAN_FILES += $$(_FILE_OPS_CIFR_DEST_)
$$(call forward-vars,$$(_FILE_OPS_CIFR_DEST_), \
	INSTALL _FILE_OPS_CIFR_MODE_)
$$(_FILE_OPS_CIFR_DEST_): $$(_FILE_OPS_CIFR_SRC_) | $$(call to-dir,$$(_FILE_OPS_CIFR_DEST_))
	$$(INSTALL) $$(call _FILE_OPS_DASH_M_,$$(_FILE_OPS_CIFR_MODE_)) "$$<" "$$@"
$$(call undefine-namespaces,_FILE_OPS_CIFR)
endef

# Creates a rule for installing a symlink. Depends on parent
# directory.
# Example: $(eval $(call _FILE_OPS_CREATE_INSTALL_FILE_RULE_,src,dest,0755))
define _FILE_OPS_CREATE_INSTALL_SYMLINK_RULE_
$$(eval $$(call _FILE_OPS_SPLIT_2_,$1,_FILE_OPS_CISR_TARGET_,_FILE_OPS_CISR_LINK_NAME_))
CLEAN_SYMLINKS += $$(_FILE_OPS_CISR_LINK_NAME_)
$$(call forward-vars,$$(_FILE_OPS_CISR_LINK_NAME_), \
	_FILE_OPS_CISR_TARGET_)
$$(_FILE_OPS_CISR_LINK_NAME_): | $$(call to-dir,$$(_FILE_OPS_CISR_LINK_NAME_))
	set -e; \
	if [ -h "$$@" ]; then \
		tgt=$$$$(readlink "$$@"); \
		if [ "$$$${tgt}" != "$$(_FILE_OPS_CISR_TARGET_)" ]; then \
			echo "'$$@' is a symlink pointing to '$$$${tgt}' instead of '$$(_FILE_OPS_CISR_TARGET_)', bailing out" >&2; \
			exit 1; \
		fi; \
	elif [ -e "$$@" ]; then \
		echo "$$@ already exists and is not a symlink, bailing out" >&2; \
		exit 1; \
	else \
		ln -s "$$(_FILE_OPS_CISR_TARGET_)" "$$@"; \
	fi
$$(call undefine-namespaces,_FILE_OPS_CISR)
endef

# Print an error if name is not a directory. To be used inside rules.
# Example $(call _FILE_OPS_BAIL_OUT_IF_NOT_DIR_,dir)
define _FILE_OPS_BAIL_OUT_IF_NOT_DIR_
if [[ ! -d "$1" ]]; then echo "$1 is not a directory, bailing out" >&2; exit 1; fi
endef

# Returns -m <foo> if foo is not a dash. Used for install invocations.
# Example: $(call _FILE_OPS_DASH_M_:0755)
define _FILE_OPS_DASH_M_
$(if $(filter-out -,$1),-m $1)
endef

define _FILE_OPS_SPLIT_2_COMMON_
$(eval _FILE_OPS_S_SPLITTED_ := $(subst :, ,$1)) \
$(eval $2 := $(word 1,$(_FILE_OPS_S_SPLITTED_))) \
$(eval $3 := $(word 2,$(_FILE_OPS_S_SPLITTED_)))
endef

define _FILE_OPS_SPLIT_2_
$(eval $(call _FILE_OPS_SPLIT_2_COMMON_,$1,$2,$3)) \
$(call undefine-namespaces,_FILE_OPS_S)
endef

define _FILE_OPS_SPLIT_3_
$(eval $(call _FILE_OPS_SPLIT_2_COMMON_,$1,$2,$3)) \
$(eval $4 := $(word 3,$(_FILE_OPS_S_SPLITTED_))) \
$(call undefine-namespaces,_FILE_OPS_S)
endef

# Special dir for storing lists of removed stuff. Sometimes the lists
# are too long for bash, so they need to be stored in files.
_FILE_OPS_DIR_ := $(BUILDDIR)/file_ops
_FILE_OPS_FILES_ := $(_FILE_OPS_DIR_)/files
_FILE_OPS_SYMLINKS_ := $(_FILE_OPS_DIR_)/symlinks
_FILE_OPS_DIRS_ := $(_FILE_OPS_DIR_)/dirs

CREATE_DIRS += $(_FILE_OPS_DIR_)
CLEAN_FILES += $(_FILE_OPS_FILES_) $(_FILE_OPS_SYMLINKS_) $(_FILE_OPS_DIRS_)

# generate rule for mkdir
$(eval $(call _FILE_OPS_CREATE_DIRS_RULE_,$(CREATE_DIRS)))

# generate rules for installing directories
$(foreach d,$(sort $(INSTALL_DIRS)), \
        $(eval $(call _FILE_OPS_CREATE_INSTALL_DIR_RULE_,$d)))

# generate rules for installing files
$(foreach f,$(sort $(INSTALL_FILES)), \
        $(eval $(call _FILE_OPS_CREATE_INSTALL_FILE_RULE_,$f)))

# generate rules for creating symlinks
$(foreach s,$(sort $(INSTALL_SYMLINKS)), \
        $(eval $(call _FILE_OPS_CREATE_INSTALL_SYMLINK_RULE_,$s)))

$(call forward-vars,_file_ops_mk_clean_, \
	_FILE_OPS_FILES_ _FILE_OPS_SYMLINKS_ _FILE_OPS_DIRS_ QUICKRMTOOL)
_file_ops_mk_clean_: $(QUICKRMTOOL_STAMP) | $(_FILE_OPS_DIR_)
	$(info writing files)
	$(file >$(_FILE_OPS_FILES_),$(words $(CLEAN_FILES)))
	$(file >>$(_FILE_OPS_FILES_),$(CLEAN_FILES))
	$(info writing symlinks)
	$(file >$(_FILE_OPS_SYMLINKS_),$(words $(CLEAN_SYMLINKS)))
	$(file >>$(_FILE_OPS_SYMLINKS_),$(CLEAN_SYMLINKS))
	$(info writing dirs)
	$(file >$(_FILE_OPS_DIRS_),$(words $(CLEAN_DIRS)))
	$(file >>$(_FILE_OPS_DIRS_),$(CLEAN_DIRS))
	set -e; \
	echo "Removing everything"; \
	"$(QUICKRMTOOL)" --files="$(_FILE_OPS_FILES_)" --symlinks="$(_FILE_OPS_SYMLINKS_)" --dirs="$(_FILE_OPS_DIRS_)"

clean: _file_ops_mk_clean_

_FILE_OPS_ALL_DIRS_ := \
	$(CREATE_DIRS) \
	$(foreach d,$(INSTALL_DIRS),$(firstword $(subst :, ,$d))) \
	$(foreach s,$(INSTALL_SYMLINKS),$(lastword $(subst :, ,$s)))

.PHONY: $(_FILE_OPS_ALL_DIRS_) _file_ops_mk_clean_

# Excluding _FILE_OPS_BAIL_OUT_IF_NOT_DIR_ and _FILE_OPS_DASH_M_
# because they are used inside recipes. Undefining them here would
# mean that inside recipes they would return empty value.
$(call undefine-namespaces,_FILE_OPS,_FILE_OPS_BAIL_OUT_IF_NOT_DIR_ _FILE_OPS_DASH_M_)
