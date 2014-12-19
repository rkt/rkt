BIN=shim.so
SRC=shim.c

$(BIN): $(SRC) Makefile
	$(CC) $(CFLAGS) $(SRC) -o $(BIN) -shared -fPIC -Wl,--no-as-needed -ldl -lc

.PHONY: clean
clean:
	rm -f $(BIN)

test:
	echo TODO
