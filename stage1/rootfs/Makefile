_SUBDIRS=usr shim diagexec prepare-app enter net-plugins net init
SUBDIRS=$(_SUBDIRS) aggregate
export CFLAGS=-Wall -Os

.PHONY: test clean subdirs $(SUBDIRS)

subdirs: $(SUBDIRS)

$(SUBDIRS):
	$(MAKE) -C $@ $(MAKECMDGOALS)

clean: $(SUBDIRS)

test: $(SUBDIRS)

aggregate: $(_SUBDIRS)
