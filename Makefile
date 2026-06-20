.PHONY: build test run tidy fmt vet lint clean install uninstall \
	install-plugin uninstall-plugin install-codex-skills uninstall-codex-skills \
	install-codex-plugin uninstall-codex-plugin

BIN := bin/schritt
PKG := ./cmd/schritt

# The skills are bundled as a single plugin (plugin/), installed into each
# runtime and invoked by name:
#   Claude Code: ~/.claude/skills/schritt  (skills-dir plugin; auto-loads as
#                                           schritt@skills-dir; /schritt:refine-pbi)
#   Codex CLI:   ~/.agents/skills          (per-skill; invoked as $refine-pbi,
#                                           via scripts/install-codex.sh)
# A symlink under ~/.claude/plugins/ is NOT auto-loaded (it needs marketplace
# registration); ~/.claude/skills/<name>/ with a plugin.json auto-loads instead.
# Override the dir with `make install-plugin CLAUDE_SKILLS_DIR=/path`.
CLAUDE_SKILLS_DIR ?= $(HOME)/.claude/skills
PLUGIN_NAME := schritt
PLUGIN_SRC := $(CURDIR)/plugin

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

# Symlink the whole plugin/ directory into ~/.claude/skills/schritt so Claude
# Code auto-loads it as the schritt@skills-dir plugin. Skills are then invoked
# as /schritt:<skill>. Restart Claude Code to pick it up; verify with
# `claude plugin list`.
install-plugin:
	@mkdir -p $(CLAUDE_SKILLS_DIR)
	@target=$(CLAUDE_SKILLS_DIR)/$(PLUGIN_NAME); \
	if [ -L $$target ]; then \
		rm -f $$target; \
	elif [ -e $$target ]; then \
		echo "skip: $$target already exists (not a symlink)"; \
		exit 0; \
	fi; \
	ln -s $(PLUGIN_SRC) $$target; \
	echo "Linked $$target -> $(PLUGIN_SRC)"; \
	echo "Restart Claude Code, then verify with: claude plugin list"

uninstall-plugin:
	@target=$(CLAUDE_SKILLS_DIR)/$(PLUGIN_NAME); \
	if [ -L $$target ]; then \
		rm -f $$target; \
		echo "Removed $$target"; \
	fi

# --- Codex install: two options ---
# (1) per-skill symlink into ~/.agents/skills. Reliable, live-edits, invoked as
#     $refine-pbi. Codex drops file-level symlinks, so the script links the skill
#     directories. This is the default/fallback.
install-codex-skills:
	@$(CURDIR)/scripts/install-codex.sh

uninstall-codex-skills:
	@$(CURDIR)/scripts/install-codex.sh --uninstall

# (2) as a Codex plugin via a local marketplace. Codex COPIES the plugin into
#     ~/.codex/plugins/cache/ (no symlink), so re-run after editing skills.
#     The marketplace lives at .agents/plugins/marketplace.json and points at
#     ./plugin. Exact `codex plugin` subcommands depend on your codex version.
PLUGIN_MARKETPLACE := schritt
install-codex-plugin:
	codex plugin marketplace add $(CURDIR)
	@echo "Added marketplace '$(PLUGIN_MARKETPLACE)'. Verify/enable with 'codex plugin', then restart codex."

uninstall-codex-plugin:
	-codex plugin marketplace remove $(PLUGIN_MARKETPLACE)
