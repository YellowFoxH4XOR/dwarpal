## MODIFIED Requirements

### Requirement: Managed instruction blocks per agent
`dwarpal agent setup <tool>` SHALL upsert a clearly-fenced managed block (begin/end markers) into the tool's project instruction file — `CLAUDE.md` for claude-code; `AGENTS.md` for codex, opencode, and pi — teaching (a) the pre-flight workflow (`check --explain-for-agent` before committing, act on `retry_hints`, never bypass hooks) and (b) how to configure Dwarpal for the repo: run `dwarpal analyze`, read the codebase, and author or update `.dwarpal.yml` consistent with both, then verify with `dwarpal rules`. The operation SHALL be idempotent: re-running replaces the managed block in place and SHALL NOT duplicate it or modify content outside the markers. Unknown tools SHALL exit 2 listing the supported ones.

#### Scenario: Fresh setup creates the instruction file
- **WHEN** `dwarpal agent setup codex` runs in a repo with no AGENTS.md
- **THEN** AGENTS.md is created containing the fenced dwarpal block, including the config-authoring instructions

#### Scenario: Existing content preserved, block replaced
- **WHEN** AGENTS.md already has user content and an older dwarpal block, and setup runs again
- **THEN** the user content is byte-identical, exactly one dwarpal block remains, with current content

#### Scenario: Unknown tool rejected
- **WHEN** `dwarpal agent setup vim` runs
- **THEN** the command exits 2 naming the supported tools
