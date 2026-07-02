## MODIFIED Requirements

### Requirement: Exit codes are a stable contract
`dwarpal check` SHALL exit 0 when no blocking findings exist, 1 when at least one blocking finding exists, and 2 on configuration or internal error. No other exit codes SHALL be emitted. This contract holds identically when `--sarif` is combined with `check`.

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
`dwarpal check --json` SHALL emit a single JSON document on stdout with the shape `{result, findings[], summary, retry_hints[]}` and nothing else on stdout. This holds whether or not `--sarif <path>` is also passed in the same invocation.

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

### Requirement: rules command lists active gates and rules
`dwarpal rules` SHALL list every enabled gate and, for Gate 3, every rule in the active rule pack, showing for each: its ID, whether it is enabled, its source (`default` or overridden via config), its severity, and (for Gate 3 rules) its tier (`regex` or `ast`).

#### Scenario: Default rule pack listed
- **WHEN** `dwarpal rules` runs in a repo with no `.dwarpal.yml`
- **THEN** the output lists all default-enabled gates and Gate 3 rules with source `default`

#### Scenario: Disabled rule shown as disabled, not omitted
- **WHEN** `gates.ai_patterns.disable_rules: ["no-hardcoded-secrets"]` is configured and `dwarpal rules` runs
- **THEN** `no-hardcoded-secrets` appears in the output marked disabled, with source naming the config file, rather than being left out of the list

### Requirement: task command declares a scope manifest
`dwarpal task "<description>" --paths <glob>[,<glob>...]` SHALL write a `.dwarpal-task.yml` at the repository root declaring the given description and path globs, for Gate 4 to consume as the task manifest.

#### Scenario: Task manifest created
- **WHEN** `dwarpal task "AUTH-42: password reset flow" --paths "src/auth/**"` runs
- **THEN** `.dwarpal-task.yml` is written containing the description and the `src/auth/**` glob, and a subsequent `dwarpal check` resolves it as the active scope manifest

#### Scenario: Missing --paths rejected
- **WHEN** `dwarpal task "AUTH-42: password reset flow"` runs without `--paths`
- **THEN** the process exits 2 with a message stating `--paths` is required
