BIN=prepare-app
SRC=prepare-app.c

$(BIN): $(SRC) Makefile install
	$(CC) $(CFLAGS) -o $@ $(SRC) -static -s
	@cp install ../aggregate/install.d/10prepare-app

.PHONY: clean
clean:
	rm -f $(BIN)

test:
	echo TODO
