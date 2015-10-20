ifneq ($(filter-out kvm,$(STAGE1_FLAVORS)),)

ASSCB_EXTRA_CFLAGS := $(RKT_DEFINES_FOR_ENTER)

STAGE1_ENTER_CMD := /enter

include stage1/makelib/aci_simple_static_c_bin.mk

endif
