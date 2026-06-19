.PHONY: build test run tidy fmt vet lint clean install uninstall

BIN := bin/schritt
PKG := ./cmd/schritt

# schritt は cgo を使わないので、クロスコンパイルや QEMU 環境での
# gcc 絡みのトラブルを避けるため既定で無効化する。
# 必要なら `make build CGO_ENABLED=1` で上書き可能。
export CGO_ENABLED ?= 0

# Override with `make install PREFIX=$HOME/.local` to avoid sudo, or
# `make install DESTDIR=/tmp/staging PREFIX=/usr/local` for packaging.
PREFIX ?= /usr/local
INSTALL_DIR := $(DESTDIR)$(PREFIX)/bin

build:
	@mkdir -p bin
	go build -o $(BIN) $(PKG)

test:
	go test ./...

run: build
	@$(BIN) $(ARGS)

tidy:
	go mod tidy

fmt:
	gofmt -w .

vet:
	go vet ./...

lint:
	golangci-lint run

clean:
	rm -rf bin/

install: build
	install -d $(INSTALL_DIR)
	install -m 0755 $(BIN) $(INSTALL_DIR)/schritt
	@echo "Installed schritt to $(INSTALL_DIR)/schritt"

uninstall:
	rm -f $(INSTALL_DIR)/schritt
	@echo "Removed schritt from $(INSTALL_DIR)"
