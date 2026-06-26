package refine

import (
	"context"

	"github.com/ystsbry/schritt/internal/agent"
)

// ClaudeRefiner runs the refinement via the `claude` CLI (Claude Code). It
// invokes the refine-pbi skill via the schritt plugin (`/schritt:refine-pbi
// <dir>`), so the plugin must be installed under ~/.claude/plugins (see
// `make install-plugin`).
type ClaudeRefiner struct {
	// Bin overrides the claude binary. Empty falls back to "claude" on PATH.
	Bin string
	// Model optionally pins a model. Empty uses the claude CLI default.
	Model string
}

func (c *ClaudeRefiner) Refine(ctx context.Context, in Input, progress func(string)) (Result, error) {
	return run(ctx, in, agent.Claude, c.Bin, c.Model, progress)
}
