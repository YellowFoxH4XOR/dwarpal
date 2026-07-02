# config-loading Specification

## Purpose
TBD - created by archiving change m0-walking-skeleton. Update Purpose after archive.
## Requirements
### Requirement: Config discovery and defaults
Dwarpal SHALL look for `.dwarpal.yml` at the git repository root. When absent, compiled-in defaults SHALL apply (mode `enforce`, diff budget: 500 lines, 20 files, 10 new files, `gates.intent_check.enabled: false`, no plugins configured). When present, its values SHALL overlay the defaults.

#### Scenario: No config file
- **WHEN** `dwarpal check` runs in a repo without `.dwarpal.yml`
- **THEN** the pipeline runs with default budgets, intent verification disabled, and no plugins, and does not error

#### Scenario: Partial config overlays defaults
- **WHEN** `.dwarpal.yml` sets only `gates.diff_budget.max_lines: 200`
- **THEN** max_lines is 200 while max_files, max_new_files, and `intent_check.enabled` remain at defaults

### Requirement: Strict schema validation
Config parsing SHALL reject unknown keys and out-of-domain values (e.g., negative budgets, unrecognized `mode`, unrecognized `intent_check.provider`, a `plugins` entry missing `exec`) with exit code 2 and a message naming the key. Misconfiguration SHALL never be silently ignored.

#### Scenario: Typo in key name
- **WHEN** `.dwarpal.yml` contains `gates.diff_budget.max_line: 100` (missing `s`)
- **THEN** `dwarpal check` exits 2 naming `max_line` as unknown

#### Scenario: Invalid mode
- **WHEN** `mode: strict` (not one of enforce|warn|ci_strict) is configured
- **THEN** `dwarpal check` exits 2 naming the invalid value

#### Scenario: Invalid intent_check provider
- **WHEN** `gates.intent_check.provider: gemini` (not one of anthropic|openai|openai-compatible) is configured
- **THEN** `dwarpal check` exits 2 naming `provider` as invalid

#### Scenario: Plugin entry missing exec
- **WHEN** a `gates.plugins` entry declares `name: semgrep` with no `exec` key
- **THEN** `dwarpal check` exits 2 naming the missing `exec` key for that plugin entry

### Requirement: Mode semantics
In `enforce` mode, error-severity findings SHALL block (exit 1). In `warn` mode, findings SHALL be reported but the exit code SHALL be 0. In `ci_strict` mode, `dwarpal bypass` SHALL be rejected in addition to standard `enforce` blocking behavior.

#### Scenario: Warn mode never blocks
- **WHEN** a diff exceeds budgets and `mode: warn` is set
- **THEN** findings are printed and the process exits 0

#### Scenario: ci_strict rejects bypass
- **WHEN** `mode: ci_strict` is set and a user runs `dwarpal bypass --reason "urgent"`
- **THEN** the bypass is rejected and the underlying blocking findings still apply

