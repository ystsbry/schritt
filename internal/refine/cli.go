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

// skillName is the refine-pbi skill's name. The agent package builds the
// runtime-specific invocation from it: "/schritt:refine-pbi" (Claude plugin)
// or "$refine-pbi" (Codex). The single plugin/skills/refine-pbi/SKILL.md is the
// source of truth for both engines.
const skillName = "refine-pbi"

// Directory sections: the skill writes one markdown file per step/scenario into
// these subdirectories of the work dir. Keep in sync with SKILL.md and store.
const (
	implementationDirName = "implementation"
	integrationDirName    = "integration_tests"
)

// singleSectionFiles maps each single-file section's filename to the Result
// field it populates. The implementation and integration sections are handled
// separately because they are directories of files. Keep filenames in sync
// with SKILL.md.
var singleSectionFiles = []struct {
	file string
	set  func(*Result, string)
}{
	{"po_questions.md", func(r *Result, s string) { r.POQuestions = s }},
	{"unit_tests.md", func(r *Result, s string) { r.UnitTests = s }},
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
	if docs, err := readDocs(filepath.Join(work, implementationDirName)); err != nil || len(docs) == 0 {
		missing = append(missing, implementationDirName+"/*.md")
	} else {
		res.Implementation = docs
	}
	if docs, err := readDocs(filepath.Join(work, integrationDirName)); err != nil || len(docs) == 0 {
		missing = append(missing, integrationDirName+"/*.md")
	} else {
		res.Integration = docs
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

これは plugin/skills/* を ~/.agents/skills/ にシンボリックリンクします
(codex はファイル単位の symlink を落とすため、ディレクトリごとリンクします)。`
	}
	return `refine-pbi skill が Claude Code に見つからない可能性があります。
リポジトリのルートで次を実行してインストールしてください:

  make install-plugin

これは plugin/ を ~/.claude/plugins/schritt にシンボリックリンクします。`
}

// readDocs reads every *.md file in dir as an ordered Doc. Files are sorted
// lexically, so the zero-padded numeric prefixes the skill uses (01-, 02-, …)
// preserve the intended order. Returns an empty slice (and nil error) when dir
// exists but has no markdown files. Used for both the implementation and the
// integration (E2E) scenario sections.
func readDocs(dir string) ([]Doc, error) {
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

	var docs []Doc
	for _, name := range names {
		raw, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return nil, err
		}
		body := normalizeBody(string(raw))
		docs = append(docs, Doc{
			File:  name,
			Title: docTitle(name, body),
			Body:  body,
		})
	}
	return docs, nil
}

// docTitle derives a human-facing label for a Doc: the text of the first
// markdown heading if present, otherwise the filename stem.
func docTitle(file, body string) string {
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
