#!/usr/bin/env bash
#
# Install schritt's skills for the OpenAI Codex CLI.
#
# Codex CLI loads Agent Skills from $HOME/.agents/skills (user-scope).
# See: https://developers.openai.com/codex/skills
#
# Each skill is installed by symlinking the whole skill DIRECTORY, not the
# SKILL.md file inside it. Codex's skill loader follows directory symlinks but
# silently drops symlinked SKILL.md files (see openai/codex#17344 / #15756),
# so a file-level symlink would never be discovered.
#
# Restart Codex after install to pick up the new skills.
#
# Usage:
#   scripts/install-codex.sh            # install (creates directory symlinks)
#   scripts/install-codex.sh --copy     # install by copy (no symlink)
#   scripts/install-codex.sh --uninstall
#
# Env:
#   AGENTS_HOME  override the agents config dir (default: ~/.agents)

set -euo pipefail

REPO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SKILLS_SRC="$REPO_DIR/plugin/skills"
AGENTS_HOME="${AGENTS_HOME:-$HOME/.agents}"
DEST_PARENT="$AGENTS_HOME/skills"

mode="symlink"
case "${1:-}" in
  --copy)      mode="copy" ;;
  --uninstall) mode="uninstall" ;;
  "")          ;;
  -h|--help)
    sed -n '2,21p' "$0"
    exit 0
    ;;
  *)
    echo "unknown option: $1" >&2
    exit 2
    ;;
esac

mkdir -p "$DEST_PARENT"

for src in "$SKILLS_SRC"/*/; do
  [ -f "$src/SKILL.md" ] || continue
  name="$(basename "$src")"
  dest="$DEST_PARENT/$name"

  if [ "$mode" = "uninstall" ]; then
    if [ -e "$dest" ] || [ -L "$dest" ]; then
      rm -rf "$dest"
      echo "removed: $dest"
    fi
    continue
  fi

  if [ -e "$dest" ] || [ -L "$dest" ]; then
    rm -rf "$dest"
  fi
  case "$mode" in
    symlink)
      ln -s "${src%/}" "$dest"
      echo "linked: $dest -> ${src%/}"
      ;;
    copy)
      cp -r "${src%/}" "$dest"
      echo "copied: ${src%/} -> $dest"
      ;;
  esac
done

if [ "$mode" != "uninstall" ]; then
  cat <<EOF

Installed schritt skills as Codex CLI skills.
Restart Codex to pick them up. schritt invokes them automatically:

  schritt refinement --engine codex   # \$refine-pbi
  schritt implement   --engine codex   # \$implement-step
EOF
fi
