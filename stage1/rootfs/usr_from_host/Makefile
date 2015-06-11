# do not include systemd in stage1, let's find it at run-time

# always run the 'install' target, even when the file is touched by the
# Makefile in the parent directory
.PHONY: clean distclean install usr.done

usr.done: Makefile install
	cp -f install ../aggregate/install.d/00usr

# prepare the install file
install: Makefile
	echo mkdir -p "\$${ROOT}" > install.tmp
	echo ln -sf usr-from-host \"\$${ROOT}/flavor\" >> install.tmp
	mv install.tmp install


distclean: clean

clean:
	rm -Rf rootfs install

test:
	echo TODO
