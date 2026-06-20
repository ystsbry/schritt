package refine

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ystsbry/schritt/internal/agent"
)

// skillName is the refine-pbi skill's name. It is invoked by this name from
// each runtime — "/refine-pbi" in Claude Code, "$refine-pbi" in Codex — so the
// single skills/refine-pbi/SKILL.md is the source of truth for both engines,
// exactly as revu drives one review-pr skill from claude and codex.
const skillName = "refine-pbi"

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

// run is the engine-agnostic refinement driver. The refine-pbi skill is
// installed into the runtime's skill directory and invoked by name (via the
// agent package); we give it a work directory containing the PBI and read the
// section files it writes back.
func run(ctx context.Context, in Input, engine, bin, model string, progress func(string)) (Result, error) {
	if in.PBINumber <= 0 {
		return Result{}, fmt.Errorf("PBINumber must be positive, got %d", in.PBINumber)
	}
	if strings.TrimSpace(in.PBIBody) == "" {
		return Result{}, errors.New("PBIBody is empty")
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

	if progress != nil {
		progress(fmt.Sprintf("%s で %s を起動中 (PBI #%d)…", engine, skillName, in.PBINumber))
	}
	var skillArgs []string
	for _, r := range repoPaths {
		skillArgs = append(skillArgs, "--repo", r)
	}
	if err := agent.Run(ctx, agent.Spec{
		Engine:    engine,
		Bin:       bin,
		Model:     model,
		SkillName: skillName,
		WorkDir:   work,
		ExtraDirs: repoPaths,
		SkillArgs: skillArgs,
	}); err != nil {
		return Result{}, err
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
			engine, strings.Join(missing, ", "), installHint(engine))
	}
	if progress != nil {
		progress("リファインメント完了")
	}
	return res, nil
}

// installHint returns the skill-install guidance for the given engine.
func installHint(engine string) string {
	if engine == agent.Codex {
		return `refine-pbi skill が Codex CLI に見つからない可能性があります。
リポジトリのルートで次を実行してインストールし、codex を再起動してください:

  scripts/install-codex.sh

これは skills/* を ~/.agents/skills/ にシンボリックリンクします
(codex はファイル単位の symlink を落とすため、ディレクトリごとリンクします)。`
	}
	return `refine-pbi skill が Claude Code に見つからない可能性があります。
リポジトリのルートで次を実行してインストールしてください:

  make install-skills

これは skills/* を ~/.claude/skills/ にシンボリックリンクします。`
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
