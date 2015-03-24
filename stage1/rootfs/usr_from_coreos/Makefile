# base just derives a base root fs from a CoreOS image downloaded and verified


usr.done: Makefile mkbase.sh cache/pxe.img manifest.d/* install
	@./mkbase.sh && touch usr.done
	@cp -f install ../aggregate/install.d/00usr

cache/pxe.img: Makefile cache.sh
	@./cache.sh

.PHONY: clean
clean:
	rm -Rf rootfs usr.squashfs usr.done manifest.txt

# TODO cleaner target for clearing cache?
test:
	echo TODO
