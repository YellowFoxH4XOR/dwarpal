## Why

The #1 adoption killer for policy tools is config maintenance — nobody
hand-writes or updates a `.dwarpal.yml`, so it rots or is never created.
Owner's insight: the developer is already inside a coding agent (Claude Code,
Codex, OpenCode, Pi) with the whole codebase in context. That agent is a
better config author than any LLM Dwarpal could call — it has full repo
context and the user already trusts and pays for it. So Dwarpal should stay
100% deterministic and offline, emit the *facts* it measures about a repo, and
let the agent author a `.dwarpal.yml` consistent with the codebase.

Config becomes a derived artifact ("set up Dwarpal for this repo"), not a
hand-maintained file.

## What Changes

- New `dwarpal analyze` (alias `dwarpal init --learn`): deterministically
  measures the repo and prints agent-consumable facts — convention fingerprint
  (naming/imports/error-idioms/function-size), a diff_budget fitted to actual
  git-history commit sizes, detected coverage artifacts and security tools to
  wire as plugins, branch-prefix and package-layering signals for
  architecture_rules. Writes nothing; it is context for the agent. `--json`
  for structured consumption.
- Agent-setup instruction blocks (CLAUDE.md / AGENTS.md) gain a "Configuring
  Dwarpal for this repo" section teaching the agent to run `dwarpal analyze`,
  read the codebase, and author/update `.dwarpal.yml` to match — then verify
  with `dwarpal rules`.
- No LLM is added to Dwarpal. Gate 7 (intent) stays as the separate,
  off-by-default, BYO-key option for CI where no agent is present.

## Capabilities

### New Capabilities
- `analyze-command`: deterministic repo analysis emitting config-authoring facts.

### Modified Capabilities
- `agent-setup`: instruction blocks add the config-authoring workflow.

## Impact

- New `internal/analyze` package + `cmd/dwarpal/analyze.go`; reuses
  `repoindex` (fingerprint) and shells to git for history sizing.
- No new dependencies; no network.

## Non-goals

- Dwarpal calling an LLM to generate config (the agent does that).
- Auto-writing `.dwarpal.yml` without the agent/human in the loop — analyze
  emits facts; the agent authors; the human skims the diff.
