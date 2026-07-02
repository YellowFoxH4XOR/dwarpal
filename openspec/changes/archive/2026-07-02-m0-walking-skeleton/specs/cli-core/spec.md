## ADDED Requirements

### Requirement: Exit codes are a stable contract
`dwarpal check` SHALL exit 0 when no blocking findings exist, 1 when at least one blocking finding exists, and 2 on configuration or internal error. No other exit codes SHALL be emitted.

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
