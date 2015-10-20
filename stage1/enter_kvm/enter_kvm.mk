ifneq ($(filter kvm,$(STAGE1_FLAVORS)),)

STAGE1_ENTER_CMD := /enter_kvm

include stage1/makelib/aci_simple_go_bin.mk

endif
