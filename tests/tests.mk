include ../makelib/*.mk

.PHONY: all

$(call make-dir-symlink,$(BUILDDIR)/bin,../bin)

PWD := $(shell pwd)

ifneq ($(RKT_STAGE1_USR_FROM),none)
all:
	@set -e; \
	HOST_SYSTEMD_VERSION=0; \
	if [ $$(basename $$(sudo readlink /proc/1/exe)) == systemd ] ; then \
		HOST_SYSTEMD_VERSION="$$(sudo /proc/1/exe --version|head -1|sed 's/^systemd \([0-9]*\)$$/\1/g')"; \
	fi; \
	STAGE1_SRC_FROM_HOST=0; \
	if tar tvf ../bin/stage1.aci rootfs/flavor|grep -q usr-from-host ; then \
		STAGE1_SRC_FROM_HOST=1; \
	fi; \
	if [ $${HOST_SYSTEMD_VERSION} -lt 220 -a $${STAGE1_SRC_FROM_HOST} -eq 1 ] ; then \
		echo "Cannot use stage1 compiled with usr-from-host because systemd >= 220 is not installed."; \
		echo "Functional tests disabled."; \
	else \
		echo [STAGE1] building prerequisites for stage1 tests...; \
		./build; \
		echo [STAGE1] starting stage1 tests...; \
		sudo GOPATH=$${GOPATH} GOROOT=$${GOROOT} $${GO} test -v $${GO_TEST_FUNC_ARGS}; \
	fi

else
all:
	@echo [STAGE1] skiping stage 1 tests as configured by user
endif
