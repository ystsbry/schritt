.PHONY: build test run tidy fmt vet lint clean install uninstall \
	install-skills uninstall-skills install-codex-skills uninstall-codex-skills

BIN := bin/schritt
PKG := ./cmd/schritt

# The refine-pbi skill is a single source of truth (skills/refine-pbi/SKILL.md)
# installed into each runtime's skill directory, then invoked by name:
#   Claude Code: ~/.claude/skills  (invoked as /refine-pbi)
#   Codex CLI:   ~/.agents/skills  (invoked as $refine-pbi, via scripts/install-codex.sh)
# Override the claude dir with `make install-skills CLAUDE_SKILLS_DIR=/path`.
CLAUDE_SKILLS_DIR ?= $(HOME)/.claude/skills
SKILLS_SRC := $(CURDIR)/skills

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

install-skills:
	@mkdir -p $(CLAUDE_SKILLS_DIR)
	@for src in $(SKILLS_SRC)/*/; do \
		name=$$(basename $$src); \
		target=$(CLAUDE_SKILLS_DIR)/$$name; \
		if [ -L $$target ]; then \
			rm -f $$target; \
		elif [ -e $$target ]; then \
			echo "skip: $$target already exists (not a symlink)"; \
			continue; \
		fi; \
		ln -s $$src $$target; \
		echo "Linked $$target -> $$src"; \
	done

uninstall-skills:
	@for src in $(SKILLS_SRC)/*/; do \
		name=$$(basename $$src); \
		target=$(CLAUDE_SKILLS_DIR)/$$name; \
		if [ -L $$target ]; then \
			rm -f $$target; \
			echo "Removed $$target"; \
		fi; \
	done

# Codex loads skills from ~/.agents/skills and needs a directory-level symlink
# (it drops file-level symlinks). The script handles that; see its header.
install-codex-skills:
	@$(CURDIR)/scripts/install-codex.sh

uninstall-codex-skills:
	@$(CURDIR)/scripts/install-codex.sh --uninstall
