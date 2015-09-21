LOCAL_UNITDIR_BASE := $(ACIROOTFSDIR)
LOCAL_UNITDIR_REST := usr/lib/systemd/system
LOCAL_UNITDIR_RESTS := \
        $(LOCAL_UNITDIR_REST) \
        $(LOCAL_UNITDIR_REST)/default.target.wants \
        $(LOCAL_UNITDIR_REST)/sockets.target.wants
LOCAL_UNITDIR := $(LOCAL_UNITDIR_BASE)/$(LOCAL_UNITDIR_REST)
LOCAL_UNIT_DIRS := $(foreach d,$(LOCAL_UNITDIR_RESTS),$(LOCAL_UNITDIR_BASE)/$d)
LOCAL_UNIT_FILES := \
        $(MK_SRCDIR)/units/default-rkt.target \
        $(MK_SRCDIR)/units/exit-watcher.service \
        $(MK_SRCDIR)/units/local-fs.target \
        $(MK_SRCDIR)/units/reaper.service \
        $(MK_SRCDIR)/units/sockets.target \
        $(MK_SRCDIR)/units/halt.target \
        $(MK_SRCDIR)/units/systemd-reboot.service \
        $(MK_SRCDIR)/units/reboot.target \
        $(MK_SRCDIR)/units/poweroff.target
LOCAL_CTRL_ALT_DEL := $(LOCAL_UNITDIR)/ctrl-alt-del.target
LOCAL_UNIT_SYMLINKS := \
        reboot.target:$(LOCAL_CTRL_ALT_DEL)
$(call setup-stamp-file,LOCAL_STAMP)

define LOCAL_SRC_TO_ACI_UNIT
$(LOCAL_UNITDIR)/$(notdir $1)
endef

LOCAL_ACI_UNIT_FILES :=
LOCAL_INSTALL_TRIPLETS :=
$(foreach u,$(LOCAL_UNIT_FILES), \
        $(eval _UNITS_MK_ACI_UNIT_ := $(call LOCAL_SRC_TO_ACI_UNIT,$u)) \
        $(eval LOCAL_ACI_UNIT_FILES += $(_UNITS_MK_ACI_UNIT_)) \
        $(eval LOCAL_INSTALL_TRIPLETS += $u:$(_UNITS_MK_ACI_UNIT_):0644))

$(call forward-vars,$(LOCAL_STAMP), \
	LOCAL_UNITDIR)
$(LOCAL_STAMP): $(LOCAL_ACI_UNIT_FILES) | $(LOCAL_UNIT_DIRS) $(LOCAL_CTRL_ALT_DEL)
	ln -sf default-rkt.target "$(LOCAL_UNITDIR)/default.target"
	touch "$@"

STAGE1_INSTALL_FILES += $(LOCAL_INSTALL_TRIPLETS)
STAGE1_INSTALL_SYMLINKS += $(LOCAL_UNIT_SYMLINKS)
STAGE1_INSTALL_DIRS += $(foreach d,$(LOCAL_UNITDIR_RESTS),$(foreach d2,$(call dir-chain,$(LOCAL_UNITDIR_BASE),$d),$(d2):0755))
STAGE1_STAMPS += $(LOCAL_STAMP)
CLEAN_SYMLINKS += $(LOCAL_UNITDIR)/default.target

$(call undefine-namespaces,LOCAL _UNITS_MK)
