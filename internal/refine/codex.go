package refine

import (
	"context"

	"github.com/ystsbry/schritt/internal/agent"
)

// CodexRefiner runs the refinement via the OpenAI `codex` CLI. It invokes the
// refine-pbi skill by name (`$refine-pbi <dir>`), so the skill must be
// installed under ~/.agents/skills (see scripts/install-codex.sh). Codex loads
// Agent Skills from ~/.agents/skills — the same single SKILL.md as claude.
type CodexRefiner struct {
	// Bin overrides the codex binary. Empty falls back to "codex" on PATH.
	Bin string
	// Model optionally pins a model. Empty uses the codex CLI default.
	Model string
	// Progress, if set, receives human-readable progress lines while the CLI
	// runs (e.g. for the TUI to surface). Optional.
	Progress func(string)
}

func (c *CodexRefiner) Refine(ctx context.Context, in Input) (Result, error) {
	return run(ctx, in, agent.Codex, c.Bin, c.Model, c.Progress)
}
