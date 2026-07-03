// Package agentsetup wires Dwarpal into coding agents (Claude Code, Codex,
// OpenCode, Pi) so the gate becomes part of the agent's loop, not just a wall
// it hits.
//
// Two mechanisms, by tool capability:
//   - Every tool reads a project instruction file (CLAUDE.md / AGENTS.md).
//     We upsert a fenced managed block teaching the pre-flight workflow:
//     run `dwarpal check --explain-for-agent` BEFORE committing, act on
//     retry_hints, never bypass hooks.
//   - Claude Code additionally supports PreToolUse hooks: we merge one into
//     .claude/settings.json so a blocking check surfaces its machine-readable
//     output to the model before the commit Bash call even runs.
//
// All writes are idempotent: managed blocks are replaced between markers
// (surrounding content untouched), and the settings merge preserves every
// existing key and never duplicates the hook.
package agentsetup

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Managed-block markers. Content between them is owned by dwarpal; everything
// outside is the user's.
const (
	beginMarker = "<!-- dwarpal:begin — managed by `dwarpal agent setup`; do not edit inside -->"
	endMarker   = "<!-- dwarpal:end -->"
)

// Tool identifies a supported coding agent.
type Tool string

const (
	ToolClaudeCode Tool = "claude-code"
	ToolCodex      Tool = "codex"
	ToolOpenCode   Tool = "opencode"
	ToolPi         Tool = "pi"
)

// SupportedTools lists valid arguments for error messages and docs.
func SupportedTools() []string {
	return []string{string(ToolClaudeCode), string(ToolCodex), string(ToolOpenCode), string(ToolPi)}
}

// instructionFile returns the tool's project instruction filename. Claude Code
// reads CLAUDE.md; Codex, OpenCode, and Pi share the AGENTS.md convention.
func instructionFile(t Tool) string {
	if t == ToolClaudeCode {
		return "CLAUDE.md"
	}
	return "AGENTS.md"
}

// agentIdentity is the value the instruction block tells the agent to export
// so provenance detection has its strongest signal.
func agentIdentity(t Tool) string {
	switch t {
	case ToolClaudeCode:
		return "Claude Code"
	case ToolCodex:
		return "Codex"
	case ToolOpenCode:
		return "OpenCode"
	case ToolPi:
		return "Pi"
	}
	return string(t)
}

// instructionBlock renders the managed block for a tool.
func instructionBlock(t Tool) string {
	return fmt.Sprintf(`%s
## Dwarpal — commit gate for this repository

This repo is gated by [Dwarpal](https://github.com/YellowFoxH4XOR/dwarpal).
Commits that violate the policy in `+"`.dwarpal.yml`"+` are blocked at the git
boundary. Work WITH the gate, not around it:

1. **Pre-flight before committing**: run `+"`dwarpal check --explain-for-agent`"+`.
   If `+"`result`"+` is `+"`blocked`"+`, read `+"`retry_hints`"+` — each is an imperative
   instruction (e.g. "Split this change: 1,240 lines exceeds the 500-line
   budget") — fix, and re-check until it passes. Only then commit.
2. **Declare intent for non-trivial tasks**: `+"`dwarpal task <TICKET-ID> --paths '<globs>'`"+`
   so the scope gate can tell intended changes from strays.
3. **Identify yourself**: export `+"`AGENTGATE_AGENT=\"%s\"`"+` in your shell (and
   keep any Co-Authored-By trailer you normally add) so provenance is exact.
4. **Never bypass**: do not use `+"`git commit --no-verify`"+`, do not edit
   `+"`.dwarpal.yml`"+` to weaken a rule, and do not add lint suppressions to make
   a check pass. If a rule is genuinely wrong here, a HUMAN approves the
   escape: `+"`DWARPAL_OVERRIDE=<rule-id>`"+` or `+"`dwarpal bypass --reason`"+`.
5. **Understand a rule**: `+"`dwarpal explain <rule-id>`"+`.

### Authoring and maintaining `+"`.dwarpal.yml`"+` (you are the config author)

Dwarpal itself never calls an LLM locally — it stays deterministic and offline.
YOU, the agent, are its judgment layer: you author and keep `+"`.dwarpal.yml`"+`
consistent with how this codebase actually works. When the user asks to "set up
Dwarpal" or "update the Dwarpal config", or when you notice the config has
drifted from reality:

1. Run `+"`dwarpal analyze --json`"+`. It measures the repo deterministically —
   languages, the commit-size distribution (with a suggested `+"`diff_budget`"+`),
   dominant import/error conventions, coverage artifacts, security tools, branch
   prefixes, and layering signals. It makes no network call and never touches
   your config or source (only the gitignored convention cache).
2. Read the codebase yourself to add what analyze cannot infer: which
   directories are true layer boundaries (→ `+"`architecture_rules`"+` forbidding,
   e.g., DB calls outside the data layer), which paths are generated (→
   `+"`diff_budget.overrides`"+`), and what a sane scope looks like.
3. Author or update `+"`.dwarpal.yml`"+` from BOTH sources. Prefer the analyze
   suggestions over generic defaults; use the distribution (not just the single
   number) to sanity-check the budget. Explain each non-obvious rule in a comment.
4. Validate: `+"`dwarpal rules`"+` prints the effective ruleset — confirm it
   matches your intent. `+"`dwarpal check`"+` must still pass on a clean tree.

Do not invent rules the codebase does not support, and do not weaken a rule
just to make a commit pass (see point 4 above).
%s`, beginMarker, agentIdentity(t), endMarker)
}

// UpsertInstructions writes/replaces the managed block in the tool's
// instruction file at root. Returns the file path and whether it was created.
func UpsertInstructions(root string, t Tool) (string, bool, error) {
	path := filepath.Join(root, instructionFile(t))
	block := instructionBlock(t)

	existing, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return path, true, os.WriteFile(path, []byte(block+"\n"), 0o644)
	}
	if err != nil {
		return path, false, err
	}

	content := string(existing)
	if i := strings.Index(content, beginMarker); i >= 0 {
		j := strings.Index(content, endMarker)
		if j < i {
			return path, false, fmt.Errorf("%s: malformed dwarpal markers (end before begin)", path)
		}
		content = content[:i] + block + content[j+len(endMarker):]
	} else {
		if !strings.HasSuffix(content, "\n") {
			content += "\n"
		}
		content += "\n" + block + "\n"
	}
	return path, false, os.WriteFile(path, []byte(content), 0o644)
}
