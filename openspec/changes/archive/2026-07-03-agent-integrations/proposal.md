## Why

Dwarpal's hooks already intercept any agent that drives git — but interception
is the *fallback* loop. The better loop is the agent pre-flighting: run
`dwarpal check --explain-for-agent` before committing, read `retry_hints`,
fix, then commit clean. That requires per-agent wiring (instruction files,
and for Claude Code a PreToolUse hook that feeds block output straight back
to the model). Today users hand-roll this; owner wants first-class support
for Claude Code, Codex, OpenCode, and Pi.

## What Changes

- New command: `dwarpal agent setup <claude-code|codex|opencode|pi>`
  - Upserts a fenced, managed instruction block into the tool's project
    instruction file (`CLAUDE.md` for Claude Code; `AGENTS.md` for Codex,
    OpenCode, Pi — the shared convention all three read). Idempotent:
    re-running replaces the block, never duplicates, never touches
    surrounding content.
  - For Claude Code additionally merges a `PreToolUse` hook into
    `.claude/settings.json` (created if absent, other keys preserved): on
    `git commit`, run `dwarpal check`; when blocked, emit the agent-readable
    JSON on stderr and exit 2 so the model receives the retry_hints *before*
    the commit attempt fails.
- Docs: one integration page per tool; README "Use inside your agent" section.

## Capabilities

### New Capabilities
- `agent-setup`: per-agent onboarding — managed instruction blocks + Claude
  Code hook merge.

### Modified Capabilities

(none — additive command; cli-core gains a subcommand but no existing
requirement changes)

## Impact

- New `internal/agentsetup` package + `cmd/dwarpal/agent.go`.
- No new dependencies (JSON merge via encoding/json).

## Non-goals

- Tool-native plugin marketplaces (a Claude Code plugin bundle can come later;
  settings-hook + CLAUDE.md achieves the same loop today).
- MCP server (tracked separately, checklist #84).
- Windows shell variants of the hook command (tracked with #75).
