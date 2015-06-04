include ../../makelib/lib.mk

.PHONY: install

ISCRIPT := $(BUILDDIR)/install.d/00usr.install

install: usr_from_host.mk
	@echo $(call dep-flavor,host) > $(ISCRIPT)
