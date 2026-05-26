BINARY  := zt
CMD     := ./cmd
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.version=$(VERSION) -s -w"

.PHONY: build install clean tidy lint

build:
	go build $(LDFLAGS) -o $(BINARY) $(CMD)

install: build
	mv $(BINARY) /usr/local/bin/$(BINARY)
	@echo "Installed to /usr/local/bin/$(BINARY)"

# Install for current user only (no sudo)
install-user: build
	mkdir -p $(HOME)/.local/bin
	mv $(BINARY) $(HOME)/.local/bin/$(BINARY)
	@echo "Installed to $(HOME)/.local/bin/$(BINARY)"
	@echo "Make sure $(HOME)/.local/bin is in your PATH"

tidy:
	go mod tidy

lint:
	golangci-lint run ./...

clean:
	rm -f $(BINARY)

# Cross-compile for common targets
release:
	GOOS=linux   GOARCH=amd64  go build $(LDFLAGS) -o dist/zt-linux-amd64   $(CMD)
	GOOS=linux   GOARCH=arm64  go build $(LDFLAGS) -o dist/zt-linux-arm64   $(CMD)
	GOOS=darwin  GOARCH=amd64  go build $(LDFLAGS) -o dist/zt-darwin-amd64  $(CMD)
	GOOS=darwin  GOARCH=arm64  go build $(LDFLAGS) -o dist/zt-darwin-arm64  $(CMD)
	GOOS=windows GOARCH=amd64  go build $(LDFLAGS) -o dist/zt-windows-amd64.exe $(CMD)
