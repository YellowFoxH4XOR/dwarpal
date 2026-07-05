package agentsetup

import (
	"fmt"
	"os"
	"path/filepath"
)

// Agent Skills: a single, portable SKILL.md that teaches the agent how to work
// with Dwarpal on demand. The format is a cross-agent open standard — every
// supported tool reads the same file; only the directory differs:
//
//   - Claude Code       → .claude/skills/dwarpal/SKILL.md
//   - Codex / OpenCode  → .agents/skills/dwarpal/SKILL.md  (the shared path
//   - Pi                   both read; Claude Code does not read .agents)
//
// The skill complements the always-on CLAUDE.md/AGENTS.md block: the block is
// durable "this repo is gated" context; the skill is the full workflow the
// agent loads (auto or via /dwarpal) when it's actually committing or setting
// up config. The invocation name is the directory name, kept stable as
// "dwarpal" across tools and releases.

// skillName is the skill directory (and thus the invocation name). Stable.
const skillName = "dwarpal"

// skillRelDir returns the tool's skills directory, relative to the repo root.
// Claude Code scans .claude/skills; Codex, OpenCode, and Pi all scan the
// shared .agents/skills convention.
func skillRelDir(t Tool) string {
	base := ".agents"
	if t == ToolClaudeCode {
		base = ".claude"
	}
	return filepath.Join(base, "skills", skillName)
}

// skillDoc renders the SKILL.md for a tool. Content is identical across tools
// except the agent identity in the AGENTGATE_AGENT line. Frontmatter is the
// universal `name`/`description` pair that every agent's parser accepts; the
// description leads with trigger phrases so auto-invocation matches when the
// agent is committing or touching config.
func skillDoc(t Tool) string {
	return fmt.Sprintf(`---
name: %s
description: >-
  Work with the Dwarpal commit gate in this repo: run the pre-flight check
  before committing and act on its retry hints, author or update .dwarpal.yml,
  and never bypass the gate. Use when committing, when a commit is blocked, or
  when setting up or adjusting Dwarpal config.
---

# Working with Dwarpal in this repository

[Dwarpal](https://github.com/YellowFoxH4XOR/dwarpal) is a deterministic commit
gate: commits that violate `+"`.dwarpal.yml`"+` are blocked at the git boundary.
Work WITH it, not around it.

## Before you commit (pre-flight)

Run `+"`dwarpal check --explain-for-agent`"+`. If `+"`result`"+` is `+"`blocked`"+`, read
`+"`retry_hints`"+` — each is an imperative fix (e.g. "Split this change: 1,240
lines exceeds the 500-line budget"). Apply them and re-run until it passes. Then
commit.

Never bypass: no `+"`git commit --no-verify`"+`, no weakening a rule in
`+"`.dwarpal.yml`"+` to pass, no lint suppressions to dodge a check. If a rule is
genuinely wrong here, a HUMAN approves the escape (`+"`DWARPAL_OVERRIDE=<rule-id>`"+`
or `+"`dwarpal bypass --reason`"+`).

## Declare intent for non-trivial work

`+"`dwarpal task <TICKET-ID> --paths '<globs>'`"+` so the scope gate can tell
intended changes from strays.

## Identify yourself

Export `+"`AGENTGATE_AGENT=\"%s\"`"+` (and keep any Co-Authored-By trailer you add)
so provenance detection is exact.

## Authoring or updating .dwarpal.yml (you are the config author)

Dwarpal never calls an LLM — every gate is deterministic and offline, and you
are its judgment layer. When asked to set up Dwarpal or adjust its config:

1. Read the codebase to set the knobs that fit it: a `+"`diff_budget`"+` sized to
   this repo's normal commits, `+"`diff_budget.overrides`"+` for generated paths,
   the protected branches, and sane scope defaults.
2. Author or update `+"`.dwarpal.yml`"+`. The compiled-in defaults are sensible —
   only override what this repo actually needs, and comment each non-obvious
   choice. Do not invent config the schema doesn't support.
3. Verify: `+"`dwarpal rules`"+` prints the effective ruleset; `+"`dwarpal check`"+`
   must still pass on a clean tree.

## Understand a rule

`+"`dwarpal explain <rule-id>`"+`.
`, skillName, agentIdentity(t))
}

// UpsertSkill writes (overwriting) the Dwarpal SKILL.md into the tool's skills
// directory at root. Returns the file path and whether it was newly created.
// Idempotent: the file is the whole identity — no registration step — so a
// re-run simply refreshes the content.
func UpsertSkill(root string, t Tool) (string, bool, error) {
	dir := filepath.Join(root, skillRelDir(t))
	path := filepath.Join(dir, "SKILL.md")

	_, statErr := os.Stat(path)
	created := os.IsNotExist(statErr)

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return path, false, err
	}
	if err := os.WriteFile(path, []byte(skillDoc(t)), 0o644); err != nil {
		return path, false, err
	}
	return path, created, nil
}
