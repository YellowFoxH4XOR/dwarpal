## MODIFIED Requirements

### Requirement: Exit codes are a stable contract
`dwarpal check` SHALL exit 0 when no blocking findings exist, 1 when at least one blocking finding exists, and 2 on configuration or internal error. No other exit codes SHALL be emitted. `dwarpal bypass` SHALL exit 0 on a recorded bypass, and exit 1 when rejected under `ci_strict` mode.

#### Scenario: Passing check
- **WHEN** `dwarpal check` runs against a staged diff within all budgets
- **THEN** the process exits 0

#### Scenario: Blocked check
- **WHEN** `dwarpal check` runs against a staged diff exceeding a budget in `enforce` mode
- **THEN** the process exits 1

#### Scenario: Invalid config
- **WHEN** `.dwarpal.yml` contains an unknown key or invalid value
- **THEN** the process exits 2 with a message naming the offending key

### Requirement: JSON output mode
`dwarpal check --json` SHALL emit a single JSON document on stdout with the shape `{result, findings[], summary, retry_hints[]}` and nothing else on stdout.

#### Scenario: Machine-readable block
- **WHEN** `dwarpal check --json` blocks a change
- **THEN** stdout parses as JSON with `result: "blocked"`, at least one finding containing `{gate, rule_id, severity, file, message}`, and at least one imperative `retry_hints` entry

#### Scenario: Human diagnostics never pollute stdout in JSON mode
- **WHEN** `dwarpal check --json` runs with any outcome
- **THEN** all human-facing diagnostics go to stderr and stdout contains only the JSON document

### Requirement: init command bootstraps a repo
`dwarpal init` SHALL detect it is inside a git work tree, write a starter `.dwarpal.yml` if none exists, install git hooks, and print each action taken. It SHALL NOT overwrite an existing `.dwarpal.yml`.

#### Scenario: Fresh repo
- **WHEN** `dwarpal init` runs in a git repo without `.dwarpal.yml`
- **THEN** `.dwarpal.yml` is created, hooks are installed, and each action is printed

#### Scenario: Existing config preserved
- **WHEN** `dwarpal init` runs in a repo that already has `.dwarpal.yml`
- **THEN** the existing file is left byte-identical and the user is told config already exists

#### Scenario: Outside a git repo
- **WHEN** `dwarpal init` runs outside any git work tree
- **THEN** the process exits 2 with a message saying a git repository is required

### Requirement: version command
`dwarpal version` SHALL print the version, commit, and build date embedded at build time.

#### Scenario: Version output
- **WHEN** `dwarpal version` runs
- **THEN** stdout contains the semantic version string

## ADDED Requirements

### Requirement: bypass command records an auditable one-shot override
`dwarpal bypass --reason "<text>"` SHALL require a non-empty `--reason`, write an auditable bypass record as both a git note on HEAD (best-effort; skipped when no commits exist) and an append-only local log (`.dwarpal/bypass.log`). Under `mode: ci_strict`, the bypass SHALL be rejected and the command SHALL exit 2 with no record written. (Consuming the record to let the next gated commit proceed is future work.)

#### Scenario: Bypass recorded in enforce mode
- **WHEN** a user runs `dwarpal bypass --reason "hotfix, reviewed by @alice"` in `enforce` mode
- **THEN** a git note is attached to HEAD, `.dwarpal/bypass.log` gains an entry containing the reason, and the process exits 0

#### Scenario: Missing reason rejected
- **WHEN** a user runs `dwarpal bypass` with no `--reason` flag
- **THEN** the command exits 2 and no bypass record is written

#### Scenario: Bypass rejected under ci_strict
- **WHEN** a user runs `dwarpal bypass --reason "urgent"` with `mode: ci_strict` configured
- **THEN** the command exits 2, no bypass record is written, and the original blocking findings remain in effect

### Requirement: doctor command reports diagnostics without mutating state
`dwarpal doctor` SHALL report, without modifying any file or git state: system git availability, git work-tree presence, `.dwarpal.yml` validity, git hook installation status (hooksPath and hook scripts), and AST language support (Go via the stdlib `go/parser` in v1). Provider/plugin reachability checks are future work.

#### Scenario: Healthy repo
- **WHEN** `dwarpal doctor` runs in a repo with valid config and installed hooks
- **THEN** it reports each check as passing and exits 0, with no files modified

#### Scenario: Hooks not installed
- **WHEN** `dwarpal doctor` runs in a repo where hooks were never installed
- **THEN** it reports the hook-status check as failing and names the remediation (`dwarpal hook install`), without installing them itself

#### Scenario: Missing plugin binary surfaced
- **WHEN** `.dwarpal.yml` configures a plugin whose `exec` binary is not on PATH
- **THEN** `dwarpal doctor` reports that plugin as unreachable
