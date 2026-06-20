package refine

import "context"

// CodexRefiner runs the refinement via the OpenAI `codex` CLI. It invokes the
// refine-pbi skill by name (`$refine-pbi <dir>`), so the skill must be
// installed under ~/.agents/skills (see scripts/install-codex.sh). Codex loads
// Agent Skills from ~/.agents/skills — the same single SKILL.md as claude.
type CodexRefiner struct {
	// Bin overrides the codex binary. Empty falls back to "codex" on PATH.
	Bin string
	// Model optionally pins a model (passed as `--model`). Empty uses the
	// codex CLI default.
	Model string
	// Progress, if set, receives human-readable progress lines while the CLI
	// runs (e.g. for the TUI to surface). Optional.
	Progress func(string)
}

func (c *CodexRefiner) Refine(ctx context.Context, in Input) (Result, error) {
	bin := c.Bin
	if bin == "" {
		bin = "codex"
	}
	return runCLI(ctx, in, cliSpec{
		name:     "codex",
		bin:      bin,
		progress: c.Progress,
		buildArgs: func(workDir string) []string {
			return codexArgs(c.Model, workDir)
		},
		installHint: codexInstallHint,
	})
}

// codexArgs builds the argv (excluding the binary) for a non-interactive
// `codex exec` run that invokes the refine-pbi skill against workDir.
//
//   - `exec` is codex's non-interactive (automation) subcommand.
//   - `--cd workDir` runs with the work dir as cwd, so `--sandbox
//     workspace-write` permits the skill to write its files there.
//   - `--skip-git-repo-check` allows running outside a git repository (the
//     work dir is a bare temp dir).
//   - the `$refine-pbi <workDir>` positional is Codex's skill-invocation
//     syntax — it resolves to ~/.agents/skills/refine-pbi.
func codexArgs(model, workDir string) []string {
	args := []string{
		"exec",
		"--cd", workDir,
		"--skip-git-repo-check",
		"--sandbox", "workspace-write",
	}
	if model != "" {
		args = append(args, "--model", model)
	}
	args = append(args, "$"+skillName+" "+workDir)
	return args
}

const codexInstallHint = `refine-pbi skill が Codex CLI に見つからない可能性があります。
リポジトリのルートで次を実行してインストールし、codex を再起動してください:

  scripts/install-codex.sh

これは skills/refine-pbi を ~/.agents/skills/ にシンボリックリンクします
(codex はファイル単位の symlink を落とすため、ディレクトリごとリンクします)。`
