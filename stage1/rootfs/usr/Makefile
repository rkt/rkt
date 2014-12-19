# base just derives a base root fs from a CoreOS image downloaded and verified


usr.done: mkbase.sh cache/pxe.img
	./mkbase.sh && touch usr.done

cache/pxe.img: cache.sh
	./cache.sh

.PHONY: clean
clean:
	rm -Rf rootfs usr.squashfs usr.done manifest.txt

# TODO cleaner target for clearing cache?
test:
	echo TODO
