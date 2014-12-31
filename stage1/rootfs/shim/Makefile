BIN=shim.so
SRC=shim.c

$(BIN): $(SRC) Makefile install
	$(CC) $(CFLAGS) $(SRC) -o $(BIN) -shared -fPIC -Wl,--no-as-needed -ldl -lc
	@cp install ../aggregate/install.d/10shim

.PHONY: clean
clean:
	rm -f $(BIN)

test:
	echo TODO
