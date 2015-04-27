PLUGINS=bin/veth bin/bridge bin/macvlan bin/host-local

../aggregate/install.d/30net-plugins: Makefile install $(PLUGINS)
	@cp install ../aggregate/install.d/30net-plugins

.PHONY: clean
clean:
	rm -f $(PLUGINS)

test:
	echo TODO
