# build systemd from source, produce an install hook selecting the needed files and host dependencies

usr.done: Makefile install
	cp -f install ../aggregate/install.d/00usr
	touch usr.done

# prepare the install file
install: Makefile host_deps.txt
	echo mkdir -p "\$${ROOT}" > install.tmp
	echo cp -af ../usr_from_src/systemd_build/installed/. \"\$${ROOT}\" >> install.tmp
	cat host_deps.txt | while read dep; do echo install -D \"$${dep}\" \"\$${ROOT}/$${dep}\"; done >> install.tmp
	echo ln -sf src \"\$${ROOT}/flavor\" >> install.tmp
	mv install.tmp install

# discover host library dependencies for all the ELF executables in systemd_build/installed, note the LD_LIBRARY_PATH= to find systemd-produced libraries.
host_deps.txt: Makefile systemd.done bash.done
	cd systemd_build/installed && find -type f | xargs file  | grep ELF | cut -f1 -d: | LD_LIBRARY_PATH=usr/lib xargs ldd | grep -v ^\\. | grep '/' | sed  -e 's/^[[:space:]]*//' -e 's/.*=> //' -e 's/ (0x[0-9a-f]*)$$//' | grep -v ^[^/] | sort -u > ../../host_deps.txt.tmp
	mv host_deps.txt.tmp host_deps.txt

# grab bash from the host
bash.done: Makefile systemd.done
	cp -af `which bash` systemd_build/installed/usr/bin/bash
	touch bash.done

# configure, build, and install systemd
systemd.done: Makefile systemd.src
	{ [ ! -e systemd_build ] || rm -Rf systemd_build; }
	mkdir systemd_build
	cd systemd_build && ../systemd/configure --disable-python-devel --disable-dbus --disable-kmod --disable-blkid --disable-chkconfig --disable-pam --disable-acl --disable-smack --disable-gcrypt --disable-elfutils --disable-libcryptsetup --disable-qrencode --disable-microhttpd --disable-gnutls --disable-binfmt --disable-vconsole --disable-readahead --disable-bootchart --disable-quotacheck --disable-tmpfiles --disable-sysusers --disable-randomseed --disable-backlight --disable-rfkill --disable-logind --disable-machined --disable-timedated --disable-timesyncd --disable-localed --disable-coredump --disable-polkit --disable-resolved --disable-networkd --disable-efi --disable-multi-seat-x --disable-myhostname --disable-gudev --disable-manpages --disable-tests --disable-blkid --disable-hibernate --disable-terminal --disable-hwdb --disable-importd && $(MAKE) && DESTDIR=$${PWD}/installed make install-strip
	touch systemd.done

# TODO(vc): it may make more sense to have the systemd source be a git submodule?
systemd.src: Makefile patches/*
	{ [ ! -e systemd ] || rm -Rf systemd; }
	mkdir systemd
	git clone --branch v215 --depth 1 git://anongit.freedesktop.org/systemd/systemd
	cd systemd && for p in ../patches/*; do patch -p1 <"$$p"; done
	cd systemd && ./autogen.sh
	touch systemd.src

.PHONY: clean distclean
clean:
	rm -Rf systemd_build systemd.done bash.done host_deps.txt rootfs usr.done install

distclean: clean
	rm -Rf systemd systemd.src

test:
	echo TODO
