# agent-setup Specification

## Purpose
TBD - created by archiving change agent-integrations. Update Purpose after archive.
## Requirements
### Requirement: Managed instruction blocks per agent
`dwarpal agent setup <tool>` SHALL upsert a clearly-fenced managed block (begin/end markers) into the tool's project instruction file — `CLAUDE.md` for claude-code; `AGENTS.md` for codex, opencode, and pi — teaching the pre-flight workflow (`check --explain-for-agent` before committing, act on `retry_hints`, never bypass hooks). The operation SHALL be idempotent: re-running replaces the managed block in place and SHALL NOT duplicate it or modify content outside the markers. Unknown tools SHALL exit 2 listing the supported ones.

#### Scenario: Fresh setup creates the instruction file
- **WHEN** `dwarpal agent setup codex` runs in a repo with no AGENTS.md
- **THEN** AGENTS.md is created containing the fenced dwarpal block

#### Scenario: Existing content preserved, block replaced
- **WHEN** AGENTS.md already has user content and an older dwarpal block, and setup runs again
- **THEN** the user content is byte-identical, exactly one dwarpal block remains, with current content

#### Scenario: Unknown tool rejected
- **WHEN** `dwarpal agent setup vim` runs
- **THEN** the command exits 2 naming the supported tools

### Requirement: Claude Code pre-flight hook merge
For claude-code, setup SHALL additionally merge a `PreToolUse` hook into `.claude/settings.json` that runs `dwarpal check` before `git commit` Bash invocations and, when the check blocks, surfaces the machine-readable output to the model (stderr + exit 2). Merging SHALL preserve all existing settings keys and existing hooks, SHALL NOT add the dwarpal hook twice, and SHALL create the file when absent.

#### Scenario: Settings created when absent
- **WHEN** setup runs with no .claude/settings.json
- **THEN** the file is created with the PreToolUse dwarpal hook

#### Scenario: Existing settings preserved and idempotent
- **WHEN** .claude/settings.json already has other keys and hooks and setup runs twice
- **THEN** the other keys and hooks remain, and exactly one dwarpal hook entry exists

