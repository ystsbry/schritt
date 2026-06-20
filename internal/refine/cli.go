package refine

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// ErrCLINotFound is returned when the chosen AI CLI is not on PATH. Callers
// surface a friendly install hint to the user.
var ErrCLINotFound = errors.New("AI CLI not found on PATH")

// skillName is the refine-pbi skill's name. It is invoked by this name from
// each runtime — "/refine-pbi" in Claude Code, "$refine-pbi" in Codex — so the
// single skills/refine-pbi/SKILL.md is the source of truth for both engines,
// exactly as revu drives one review-pr skill from claude and codex.
const skillName = "refine-pbi"

// skillInvocation builds the by-name skill call passed to a runtime. prefix is
// the runtime's skill-invocation sigil ("/" for Claude Code, "$" for Codex).
// The work dir is the positional argument; each repo (if any) is passed as a
// repeated --repo flag the skill recognises (see SKILL.md).
func skillInvocation(prefix, workDir string, repoPaths []string) string {
	s := prefix + skillName + " " + workDir
	for _, r := range repoPaths {
		s += " --repo " + r
	}
	return s
}

// implementationDirName is the subdirectory of the work dir into which the
// skill writes one markdown file per implementation step. Keep it in sync with
// SKILL.md's "出力" section and store.implementationSubdir.
const implementationDirName = "implementation"

// singleSectionFiles maps each single-file section's filename to the Result
// field it populates. The implementation section is handled separately because
// it is a directory of step files. Keep filenames in sync with SKILL.md.
var singleSectionFiles = []struct {
	file string
	set  func(*Result, string)
}{
	{"po_questions.md", func(r *Result, s string) { r.POQuestions = s }},
	{"unit_tests.md", func(r *Result, s string) { r.UnitTests = s }},
	{"integration_tests.md", func(r *Result, s string) { r.IntegrationTests = s }},
}

// cliSpec describes how to drive one CLI engine (claude, codex, …). The shared
// runner handles everything else: the work directory, running the process, and
// reading the section files back.
type cliSpec struct {
	// name is the engine name used in progress/error messages.
	name string
	// bin is the resolved executable (already defaulted by the refiner).
	bin string
	// progress, if set, receives human-readable progress lines.
	progress func(string)
	// buildArgs returns the argv (excluding bin) to run. workDir is the
	// directory holding pbi.md/notes.md that the skill must read and write
	// its output into; it is passed to the skill as its argument. repoPaths
	// are the target repositories to grant read access to, or empty if none.
	buildArgs func(workDir string, repoPaths []string) []string
	// installHint is shown when the skill appears not to be installed (the
	// run produced no section files).
	installHint string
}

// runCLI is the engine-agnostic refinement driver. We don't pass the skill
// text as a prompt: the skill is installed into the runtime's skill directory
// (~/.claude/skills or ~/.agents/skills) and invoked by name. We give it a
// work directory containing the PBI and read the section files it writes back.
func runCLI(ctx context.Context, in Input, spec cliSpec) (Result, error) {
	if in.PBINumber <= 0 {
		return Result{}, fmt.Errorf("PBINumber must be positive, got %d", in.PBINumber)
	}
	if strings.TrimSpace(in.PBIBody) == "" {
		return Result{}, errors.New("PBIBody is empty")
	}
	if _, err := exec.LookPath(spec.bin); err != nil {
		return Result{}, fmt.Errorf("%w (%s)", ErrCLINotFound, spec.bin)
	}

	var repoPaths []string
	for _, raw := range in.RepoPaths {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		abs, err := filepath.Abs(raw)
		if err != nil {
			return Result{}, fmt.Errorf("resolve repo path %q: %w", raw, err)
		}
		st, err := os.Stat(abs)
		if err != nil {
			return Result{}, fmt.Errorf("repo path %q: %w", raw, err)
		}
		if !st.IsDir() {
			return Result{}, fmt.Errorf("repo path %q is not a directory", raw)
		}
		repoPaths = append(repoPaths, abs)
	}

	work, err := os.MkdirTemp("", "schritt-refine-*")
	if err != nil {
		return Result{}, fmt.Errorf("create work dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(work) }()

	if err := os.WriteFile(filepath.Join(work, "pbi.md"), []byte(in.PBIBody), 0o644); err != nil {
		return Result{}, fmt.Errorf("write pbi.md: %w", err)
	}
	if notes := strings.TrimSpace(in.Notes); notes != "" {
		if err := os.WriteFile(filepath.Join(work, "notes.md"), []byte(in.Notes), 0o644); err != nil {
			return Result{}, fmt.Errorf("write notes.md: %w", err)
		}
	}

	if spec.progress != nil {
		spec.progress(fmt.Sprintf("%s で $%s を起動中 (PBI #%d)…", spec.name, skillName, in.PBINumber))
	}
	cmd := exec.CommandContext(ctx, spec.bin, spec.buildArgs(work, repoPaths)...)
	// Non-interactive: no stdin passthrough. Capture stderr for diagnostics;
	// stdout is the agent's prose, which we ignore — the real output is the
	// section files it writes into the work dir.
	var stderr strings.Builder
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return Result{}, ctx.Err()
		}
		return Result{}, fmt.Errorf("%s run failed: %w\n%s", spec.name, err, stderr.String())
	}

	var res Result
	var missing []string
	for _, sf := range singleSectionFiles {
		body, err := os.ReadFile(filepath.Join(work, sf.file))
		if err != nil {
			missing = append(missing, sf.file)
			continue
		}
		sf.set(&res, normalizeBody(string(body)))
	}
	steps, err := readImplementationSteps(filepath.Join(work, implementationDirName))
	if err != nil || len(steps) == 0 {
		missing = append(missing, implementationDirName+"/*.md")
	} else {
		res.Implementation = steps
	}
	if len(missing) > 0 {
		// The most common cause is that the skill is not installed for this
		// runtime, so the by-name invocation produced nothing.
		return Result{}, fmt.Errorf("%s が期待するセクションファイルを生成しませんでした (%s)。\n%s",
			spec.name, strings.Join(missing, ", "), spec.installHint)
	}
	if spec.progress != nil {
		spec.progress("リファインメント完了")
	}
	return res, nil
}

// readImplementationSteps reads every *.md file in dir as an ordered
// implementation step. Files are sorted lexically, so the zero-padded numeric
// prefixes the skill uses (01-, 02-, …) preserve the intended order. Returns
// an empty slice (and nil error) when dir exists but has no markdown files.
func readImplementationSteps(dir string) ([]ImplementationStep, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.EqualFold(filepath.Ext(e.Name()), ".md") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)

	var steps []ImplementationStep
	for _, name := range names {
		raw, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return nil, err
		}
		body := normalizeBody(string(raw))
		steps = append(steps, ImplementationStep{
			File:  name,
			Title: stepTitle(name, body),
			Body:  body,
		})
	}
	return steps, nil
}

// stepTitle derives a human-facing label for an implementation step: the text
// of the first markdown heading if present, otherwise the filename stem.
func stepTitle(file, body string) string {
	for _, line := range strings.Split(body, "\n") {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "# ") {
			return strings.TrimSpace(t[2:])
		}
	}
	return strings.TrimSuffix(file, filepath.Ext(file))
}

// normalizeBody trims trailing blank lines and ensures a single trailing
// newline, so stored markdown is consistent regardless of how the AI wrote it.
func normalizeBody(s string) string {
	return strings.TrimRight(s, "\n") + "\n"
}
