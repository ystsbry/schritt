// Package agent runs one by-name skill invocation against an AI CLI engine
// (Claude Code or OpenAI Codex). It owns the engine-specific argv construction
// (sandbox flags, directory grants, the /name vs $name invocation sigil) so
// each pipeline stage — refine, implement, … — shares one source of truth for
// how a skill is launched.
//
// A stage writes its inputs into a work directory, calls Run with a Spec, then
// reads back whatever files the skill produced. Run only launches the process
// and reports failure; it does not know about a stage's input/output files.
package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

// ErrCLINotFound is returned when the chosen engine's executable is not on
// PATH. Callers surface a friendly install hint to the user.
var ErrCLINotFound = errors.New("AI CLI not found on PATH")

// Engine identifies which CLI runs the skill.
const (
	Claude = "claude"
	Codex  = "codex"
)

// Spec is a single by-name skill invocation.
type Spec struct {
	// Engine selects the CLI (Claude or Codex). Required.
	Engine string
	// Bin overrides the engine binary. Empty falls back to the engine default.
	Bin string
	// Model optionally pins a model (passed as --model). Empty uses the CLI
	// default.
	Model string
	// SkillName is the installed skill invoked by name: "/<name>" for Claude,
	// "$<name>" for Codex. Required.
	SkillName string
	// WorkDir is the skill's primary directory: the positional argument and a
	// writable grant (cwd for Codex, the first --add-dir for Claude). Required.
	WorkDir string
	// ExtraDirs are additional writable directory grants — e.g. target
	// repositories the skill reads or modifies (--add-dir).
	ExtraDirs []string
	// SkillArgs are appended to the invocation after WorkDir, verbatim, e.g.
	// ["--repo", "/path"]. The caller composes them.
	SkillArgs []string
	// MCPServers are MCP servers to make available to the agent — e.g. a
	// browser-over-CDP server for the verify stage. Empty for stages that
	// don't need tools beyond the built-ins (refine, implement).
	MCPServers []MCPServer
	// AllowedTools auto-approves the listed tool patterns in non-interactive
	// mode (Claude --allowedTools), e.g. ["mcp__chrome-devtools"]. Needed so
	// MCP tool calls aren't blocked when there is no TTY to prompt.
	AllowedTools []string
	// NetworkAccess opens outbound network egress for the run. Only meaningful
	// for Codex, whose workspace-write sandbox blocks egress by default;
	// browser verification needs it to reach the app URL.
	NetworkAccess bool
	// Stdout, if set, receives the CLI's stdout (the agent's prose) live. Nil
	// discards it. Stderr, if set, receives stderr live; nil captures it and
	// includes it in the returned error.
	Stdout io.Writer
	Stderr io.Writer
}

// MCPServer describes a stdio MCP server to expose to the agent. The agent's
// tools from it are named mcp__<Name>__<tool>.
type MCPServer struct {
	Name    string   // logical name, e.g. "chrome-devtools"
	Command string   // executable, e.g. "npx"
	Args    []string // e.g. ["-y", "chrome-devtools-mcp@latest"]
}

// DefaultBin returns the default executable name for an engine.
func DefaultBin(engine string) string {
	switch engine {
	case Codex:
		return "codex"
	default:
		return "claude"
	}
}

// bin returns the resolved binary for the spec.
func (s Spec) bin() string {
	if s.Bin != "" {
		return s.Bin
	}
	return DefaultBin(s.Engine)
}

// Invocation returns the by-name skill call string passed to the engine, e.g.
// "/refine-pbi /work --repo /repo" (Claude) or "$implement-step /work" (Codex).
func (s Spec) Invocation() string {
	sigil := "/"
	if s.Engine == Codex {
		sigil = "$"
	}
	inv := sigil + s.SkillName + " " + s.WorkDir
	for _, a := range s.SkillArgs {
		inv += " " + a
	}
	return inv
}

// Args returns the full argv (excluding the binary) for the invocation.
func (s Spec) Args() []string {
	switch s.Engine {
	case Codex:
		// `exec` is codex's non-interactive subcommand. `--cd WorkDir` sets the
		// cwd so `--sandbox workspace-write` permits writes there;
		// `--skip-git-repo-check` allows the bare temp work dir; each ExtraDir
		// is granted with its own `--add-dir`. The "$name ..." positional is
		// codex's skill-invocation syntax.
		args := []string{
			"exec",
			"--cd", s.WorkDir,
			"--skip-git-repo-check",
			"--sandbox", "workspace-write",
		}
		for _, d := range s.ExtraDirs {
			args = append(args, "--add-dir", d)
		}
		// Per-invocation config overrides (-c). network_access opens egress for
		// browser verification; mcp_servers registers any MCP servers.
		if s.NetworkAccess {
			args = append(args, "-c", "sandbox_workspace_write.network_access=true")
		}
		for _, m := range s.MCPServers {
			args = append(args, "-c", fmt.Sprintf("mcp_servers.%s.command=%q", m.Name, m.Command))
			if len(m.Args) > 0 {
				js, _ := json.Marshal(m.Args)
				args = append(args, "-c", fmt.Sprintf("mcp_servers.%s.args=%s", m.Name, string(js)))
			}
		}
		if s.Model != "" {
			args = append(args, "--model", s.Model)
		}
		return append(args, s.Invocation())
	default: // Claude
		// `--add-dir` is variadic (WorkDir + ExtraDirs), placed before
		// `--permission-mode` so the next flag terminates the capture and the
		// `--print` prompt isn't swallowed. acceptEdits auto-approves writes in
		// non-interactive mode.
		var args []string
		if s.Model != "" {
			args = append(args, "--model", s.Model)
		}
		args = append(args, "--add-dir", s.WorkDir)
		args = append(args, s.ExtraDirs...)
		args = append(args, "--permission-mode", "acceptEdits")
		if len(s.MCPServers) > 0 {
			args = append(args, "--mcp-config", claudeMCPConfig(s.MCPServers))
		}
		if len(s.AllowedTools) > 0 {
			args = append(args, "--allowedTools", strings.Join(s.AllowedTools, ","))
		}
		return append(args, "--print", s.Invocation())
	}
}

// claudeMCPConfig renders the MCP servers as the JSON Claude Code's
// --mcp-config accepts inline.
func claudeMCPConfig(servers []MCPServer) string {
	type srv struct {
		Command string   `json:"command"`
		Args    []string `json:"args,omitempty"`
	}
	m := map[string]srv{}
	for _, s := range servers {
		m[s.Name] = srv{Command: s.Command, Args: s.Args}
	}
	b, _ := json.Marshal(struct {
		MCPServers map[string]srv `json:"mcpServers"`
	}{MCPServers: m})
	return string(b)
}

// Run launches the skill invocation and waits for it to finish. It returns
// ErrCLINotFound if the engine binary is missing, ctx.Err() on cancellation,
// or a wrapped error (including captured stderr when Spec.Stderr is nil) on a
// non-zero exit.
func Run(ctx context.Context, s Spec) error {
	if s.Engine != Claude && s.Engine != Codex {
		return fmt.Errorf("unknown engine %q", s.Engine)
	}
	if s.SkillName == "" || s.WorkDir == "" {
		return errors.New("agent: SkillName and WorkDir are required")
	}
	bin := s.bin()
	if _, err := exec.LookPath(bin); err != nil {
		return fmt.Errorf("%w (%s)", ErrCLINotFound, bin)
	}

	cmd := exec.CommandContext(ctx, bin, s.Args()...)
	var captured strings.Builder
	cmd.Stdout = s.Stdout
	if s.Stderr != nil {
		cmd.Stderr = s.Stderr
	} else {
		cmd.Stderr = &captured
	}
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if captured.Len() > 0 {
			return fmt.Errorf("%s run failed: %w\n%s", s.Engine, err, captured.String())
		}
		return fmt.Errorf("%s run failed: %w", s.Engine, err)
	}
	return nil
}
