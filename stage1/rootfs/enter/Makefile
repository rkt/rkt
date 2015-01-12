BIN=enter
SRC=enter.c

$(BIN): $(SRC) Makefile install
	$(CC) $(CFLAGS) -o $@ $(SRC) -static -s
	@cp install ../aggregate/install.d/10enter

.PHONY: clean
clean:
	rm -f $(BIN)

test:
	echo TODO
