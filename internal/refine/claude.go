package refine

import (
	"context"

	"github.com/ystsbry/schritt/internal/agent"
)

// ClaudeRefiner runs the refinement via the `claude` CLI (Claude Code). It
// invokes the refine-pbi skill by name (`/refine-pbi <dir>`), so the skill must
// be installed under ~/.claude/skills (see `make install-skills`).
type ClaudeRefiner struct {
	// Bin overrides the claude binary. Empty falls back to "claude" on PATH.
	Bin string
	// Model optionally pins a model. Empty uses the claude CLI default.
	Model string
	// Progress, if set, receives human-readable progress lines while the CLI
	// runs (e.g. for the TUI to surface). Optional.
	Progress func(string)
}

func (c *ClaudeRefiner) Refine(ctx context.Context, in Input) (Result, error) {
	return run(ctx, in, agent.Claude, c.Bin, c.Model, c.Progress)
}
