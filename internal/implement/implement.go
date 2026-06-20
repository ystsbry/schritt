// Package implement is the second pipeline stage: implementing the refinement's
// plan one step at a time. For each implementation step it drives the
// implement-step skill (via the agent package) to write code into the target
// repository and produce a per-step report of what was implemented and which
// unit tests were written.
package implement

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/ystsbry/schritt/internal/agent"
)

// skillName is the implement-step skill, invoked by name from each runtime.
const skillName = "implement-step"

// Input is one implementation step to carry out.
type Input struct {
	// StepTitle is the step's human label (used in messages/report fallback).
	StepTitle string
	// StepBody is the step's plan markdown, written to step.md for the skill.
	StepBody string
	// PBIBody is the original PBI markdown for context (optional).
	PBIBody string
	// Notes is supplementary context (optional).
	Notes string
	// RepoPaths are the target repositories the step is implemented in. At
	// least one is required for a real run.
	RepoPaths []string
}

// Result holds the per-step report produced by the skill.
type Result struct {
	// Report is the markdown report (what was implemented + tests written).
	Report string
}

// Implementer carries out one implementation step.
type Implementer interface {
	Implement(ctx context.Context, in Input) (Result, error)
}

// reportFile is the file the skill writes its report to, inside the work dir.
const reportFile = "report.md"

// runStep is the engine-agnostic driver: write the step inputs into a work
// dir, invoke the implement-step skill by name, and read the report back.
// stream, if set, receives the agent's live output (so a CLI can show progress
// while the agent edits the repo).
func runStep(ctx context.Context, in Input, engine, bin, model string, stream io.Writer) (Result, error) {
	if strings.TrimSpace(in.StepBody) == "" {
		return Result{}, errors.New("StepBody is empty")
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
	if len(repoPaths) == 0 {
		return Result{}, errors.New("少なくとも1つのリポジトリ(--repo)が必要です")
	}

	work, err := os.MkdirTemp("", "schritt-implement-*")
	if err != nil {
		return Result{}, fmt.Errorf("create work dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(work) }()

	if err := os.WriteFile(filepath.Join(work, "step.md"), []byte(in.StepBody), 0o644); err != nil {
		return Result{}, fmt.Errorf("write step.md: %w", err)
	}
	if b := strings.TrimSpace(in.PBIBody); b != "" {
		if err := os.WriteFile(filepath.Join(work, "pbi.md"), []byte(in.PBIBody), 0o644); err != nil {
			return Result{}, fmt.Errorf("write pbi.md: %w", err)
		}
	}
	if b := strings.TrimSpace(in.Notes); b != "" {
		if err := os.WriteFile(filepath.Join(work, "notes.md"), []byte(in.Notes), 0o644); err != nil {
			return Result{}, fmt.Errorf("write notes.md: %w", err)
		}
	}

	var skillArgs []string
	for _, r := range repoPaths {
		skillArgs = append(skillArgs, "--repo", r)
	}
	err = agent.Run(ctx, agent.Spec{
		Engine:    engine,
		Bin:       bin,
		Model:     model,
		SkillName: skillName,
		WorkDir:   work,
		ExtraDirs: repoPaths,
		SkillArgs: skillArgs,
		Stdout:    stream,
		Stderr:    stream,
	})
	if err != nil {
		return Result{}, err
	}

	body, err := os.ReadFile(filepath.Join(work, reportFile))
	if err != nil {
		return Result{}, fmt.Errorf("%s が %s を生成しませんでした。skill 未インストールの可能性があります。\n%s",
			engine, reportFile, installHint(engine))
	}
	return Result{Report: strings.TrimRight(string(body), "\n") + "\n"}, nil
}

// installHint returns the skill-install guidance for the given engine.
func installHint(engine string) string {
	if engine == agent.Codex {
		return `implement-step skill が Codex CLI に見つからない可能性があります。
リポジトリのルートで scripts/install-codex.sh を実行し、codex を再起動してください。`
	}
	return `implement-step skill が Claude Code に見つからない可能性があります。
リポジトリのルートで make install-skills を実行してください。`
}
