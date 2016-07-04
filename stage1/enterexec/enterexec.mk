ASSCB_EXTRA_CFLAGS := -I$(MK_SRCDIR)/../diagnostic
ASSCB_EXTRA_HEADERS := ../diagnostic/diagnostic-util.h
ASSCB_EXTRA_SOURCES := ../diagnostic/diagnostic-util.c

include stage1/makelib/aci_simple_static_c_bin.mk
