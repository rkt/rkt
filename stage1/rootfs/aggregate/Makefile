S1=s1rootfs
S1TAR=$(S1).tar

../../../stage0/stage1_rootfs/bin.go: $(S1TAR) Makefile
	@mkdir -p $$(dirname $@)
	TMP=$$(mktemp -d -t stage1-XXXXXX)				&& \
	ln -s "$(CURDIR)/$(S1TAR)" "$$TMP"				&& \
	go-bindata -o $@ -prefix="$$TMP" -pkg=stage1_rootfs "$$TMP"	&& \
	rm -Rf "$$TMP"

$(S1TAR): aggregate.sh Makefile scripts/* units/*
	@./aggregate.sh && tar cf $(S1TAR) -C $(S1) .

.PHONY: clean
clean:
	rm -Rf $(S1) $(S1TAR)

test:
	echo TODO
