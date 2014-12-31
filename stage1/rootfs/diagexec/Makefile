BIN=diagexec
SRC=diagexec.c

$(BIN): $(SRC) elf.h Makefile install
	$(CC) $(CFLAGS) -o $@ $(SRC) -static -s
	@cp install ../aggregate/install.d/10diagexec

.PHONY: clean
clean:
	rm -f $(BIN)

test:
	echo TODO
