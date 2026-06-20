package refine

import "context"

// ClaudeRefiner runs the refinement via the `claude` CLI (Claude Code). It
// invokes the refine-pbi skill by name (`/refine-pbi <dir>`), so the skill must
// be installed under ~/.claude/skills (see `make install-skills`).
type ClaudeRefiner struct {
	// Bin overrides the claude binary. Empty falls back to "claude" on PATH.
	Bin string
	// Model optionally pins a model (passed as `--model`). Empty uses the
	// claude CLI default.
	Model string
	// Progress, if set, receives human-readable progress lines while the CLI
	// runs (e.g. for the TUI to surface). Optional.
	Progress func(string)
}

func (c *ClaudeRefiner) Refine(ctx context.Context, in Input) (Result, error) {
	bin := c.Bin
	if bin == "" {
		bin = "claude"
	}
	return runCLI(ctx, in, cliSpec{
		name:     "claude",
		bin:      bin,
		progress: c.Progress,
		buildArgs: func(workDir string) []string {
			return claudeArgs(c.Model, workDir)
		},
		installHint: claudeInstallHint,
	})
}

// claudeArgs builds the argv (excluding the binary) for a non-interactive
// `claude --print` run that invokes the refine-pbi skill against workDir.
//
//   - `--add-dir workDir` grants the run read/write access to the work dir.
//   - `--permission-mode acceptEdits` auto-approves writes (no TTY to prompt).
//   - `--print "/refine-pbi <workDir>"` invokes the skill by name once.
//
// `--add-dir` is variadic, so it is placed before `--print` to avoid swallowing
// the prompt; `--model` (if any) goes first.
func claudeArgs(model, workDir string) []string {
	var args []string
	if model != "" {
		args = append(args, "--model", model)
	}
	args = append(args,
		"--add-dir", workDir,
		"--permission-mode", "acceptEdits",
		"--print", "/"+skillName+" "+workDir,
	)
	return args
}

const claudeInstallHint = `refine-pbi skill が Claude Code に見つからない可能性があります。
リポジトリのルートで次を実行してインストールしてください:

  make install-skills

これは skills/refine-pbi を ~/.claude/skills/ にシンボリックリンクします。`
