## MODIFIED Requirements

### Requirement: Exit codes are a stable contract
`dwarpal check` SHALL exit 0 when no blocking findings exist, 1 when at
least one blocking finding exists, and 2 on configuration or internal
error. No other exit codes SHALL be emitted. `dwarpal explain` SHALL follow
the same contract: 0 on a successful lookup, 2 when the finding id is
unrecognized or config is invalid.

#### Scenario: Passing check
- **WHEN** `dwarpal check` runs against a staged diff within all budgets
- **THEN** the process exits 0

#### Scenario: Blocked check
- **WHEN** `dwarpal check` runs against a staged diff exceeding a budget in
  `enforce` mode
- **THEN** the process exits 1

#### Scenario: Invalid config
- **WHEN** `.dwarpal.yml` contains an unknown key or invalid value
- **THEN** the process exits 2 with a message naming the offending key

#### Scenario: explain exits 0 on known id
- **WHEN** `dwarpal explain <finding-id>` runs with a recognized id
- **THEN** the process exits 0

#### Scenario: explain exits 2 on unknown id
- **WHEN** `dwarpal explain <finding-id>` runs with an unrecognized id
- **THEN** the process exits 2

### Requirement: JSON output mode
`dwarpal check --json` SHALL emit a single JSON document on stdout with the
shape `{result, findings[], summary, retry_hints[]}` and nothing else on
stdout. `retry_hints` SHALL be an array index-aligned with `findings` —
`retry_hints[i]` SHALL be one imperative, machine-consumable remediation
instruction for `findings[i]`, populated by every finding-producing gate
(diff-budget, branch-policy, ai-patterns, scope, diff-coverage, convention-
drift). `retry_hints` and `findings` SHALL always be equal length.

#### Scenario: Machine-readable block
- **WHEN** `dwarpal check --json` blocks a change
- **THEN** stdout parses as JSON with `result: "blocked"`, at least one
  finding containing `{gate, rule_id, severity, file, message}`, and at
  least one imperative `retry_hints` entry

#### Scenario: Human diagnostics never pollute stdout in JSON mode
- **WHEN** `dwarpal check --json` runs with any outcome
- **THEN** all human-facing diagnostics go to stderr and stdout contains
  only the JSON document

#### Scenario: retry_hints is index-aligned with findings
- **WHEN** `dwarpal check --json` produces three findings from different
  gates (e.g. one diff-budget, one coverage, one drift)
- **THEN** `retry_hints` has exactly three entries, and `retry_hints[i]`
  is the imperative fix instruction for `findings[i]` specifically, not a
  generic per-gate summary

### Requirement: init command bootstraps a repo
`dwarpal init` SHALL detect it is inside a git work tree, write a starter
`.dwarpal.yml` if none exists, install git hooks, and print each action
taken. It SHALL NOT overwrite an existing `.dwarpal.yml`.

#### Scenario: Fresh repo
- **WHEN** `dwarpal init` runs in a git repo without `.dwarpal.yml`
- **THEN** `.dwarpal.yml` is created, hooks are installed, and each action
  is printed

#### Scenario: Existing config preserved
- **WHEN** `dwarpal init` runs in a repo that already has `.dwarpal.yml`
- **THEN** the existing file is left byte-identical and the user is told
  config already exists

#### Scenario: Outside a git repo
- **WHEN** `dwarpal init` runs outside any git work tree
- **THEN** the process exits 2 with a message saying a git repository is
  required

### Requirement: version command
`dwarpal version` SHALL print the version, commit, and build date embedded
at build time.

#### Scenario: Version output
- **WHEN** `dwarpal version` runs
- **THEN** stdout contains the semantic version string

## ADDED Requirements

### Requirement: explain command is registered in the CLI surface
`dwarpal explain <finding-id>` SHALL be a top-level command alongside
`check`, `init`, `rules`, and `version`, visible in `dwarpal --help`, and
SHALL delegate its lookup and output behavior to the `explain-command`
capability.

#### Scenario: Listed in help
- **WHEN** `dwarpal --help` runs
- **THEN** the output lists `explain` among the available commands with a
  one-line description

#### Scenario: Missing argument
- **WHEN** `dwarpal explain` runs with no `<finding-id>` argument
- **THEN** the process exits 2 with a usage message naming the required
  argument
