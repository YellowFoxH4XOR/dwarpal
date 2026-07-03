# analyze-command Specification

## Purpose
TBD - created by archiving change agent-config-authoring. Update Purpose after archive.
## Requirements
### Requirement: Deterministic repo analysis for config authoring
`dwarpal analyze` SHALL measure the repository without any network call or LLM and print facts an agent can use to author a `.dwarpal.yml` consistent with the codebase: the per-language convention fingerprint (naming case, import style, error idiom, average function size), a suggested diff budget derived from the repository's own recent commit-size distribution, detected coverage artifacts, detected security tools (e.g. gitleaks, semgrep) suitable as plugin gates, observed branch-name prefixes, and candidate architecture-rule layering signals. It SHALL write no config or source files; its only permitted side effect is warming the gitignored convention cache (`.dwarpal/cache/`) that the gates already use.

#### Scenario: Analyze emits measured facts
- **WHEN** `dwarpal analyze` runs in a repo with commit history and source files
- **THEN** it prints a suggested diff budget reflecting the repo's actual commit sizes and the dominant conventions per detected language, and creates no files outside the gitignored `.dwarpal/cache/`

#### Scenario: JSON output for agents
- **WHEN** `dwarpal analyze --json` runs
- **THEN** stdout is a single JSON document of the measured facts and nothing else

#### Scenario: No network
- **WHEN** `dwarpal analyze` runs with no network access and no API key configured
- **THEN** it completes successfully — analysis is purely local

