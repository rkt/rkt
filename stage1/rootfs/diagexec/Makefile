BIN=diagexec
SRC=diagexec.c

$(BIN): $(SRC) elf.h Makefile
	$(CC) $(CFLAGS) -o $@ $(SRC) -static -s

.PHONY: clean
clean:
	rm -f $(BIN)

test:
	echo TODO
