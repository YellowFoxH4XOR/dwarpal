## MODIFIED Requirements

### Requirement: Config discovery and defaults
Dwarpal SHALL look for `.dwarpal.yml` at the git repository root. When absent, compiled-in defaults SHALL apply (mode `enforce`, diff budget: 500 lines, 20 files, 10 new files, `provenance.apply_gates_to: agent-only`, `gates.ai_patterns.enabled: true`, `gates.scope.require_task_manifest: false`). When present, its values SHALL overlay the defaults.

#### Scenario: No config file
- **WHEN** `dwarpal check` runs in a repo without `.dwarpal.yml`
- **THEN** the pipeline runs with default budgets, default provenance settings, and default gate settings, and does not error

#### Scenario: Partial config overlays defaults
- **WHEN** `.dwarpal.yml` sets only `gates.diff_budget.max_lines: 200`
- **THEN** max_lines is 200 while max_files, max_new_files, and all M1 gate settings remain at defaults

#### Scenario: Provenance and gate sections overlay independently
- **WHEN** `.dwarpal.yml` sets only `provenance.apply_gates_to: all-commits`
- **THEN** that value is used while `gates.ai_patterns`, `gates.scope`, and `gates.branch_policy` remain at their defaults

### Requirement: Strict schema validation
Config parsing SHALL reject unknown keys and out-of-domain values (e.g., negative budgets, unrecognized `mode`, unrecognized `provenance.apply_gates_to` value, unknown rule ID in `gates.ai_patterns.disable_rules`) with exit code 2 and a message naming the key. Misconfiguration SHALL never be silently ignored.

#### Scenario: Typo in key name
- **WHEN** `.dwarpal.yml` contains `gates.diff_budget.max_line: 100` (missing `s`)
- **THEN** `dwarpal check` exits 2 naming `max_line` as unknown

#### Scenario: Invalid mode
- **WHEN** `mode: strict` (not one of enforce|warn|ci_strict) is configured
- **THEN** `dwarpal check` exits 2 naming the invalid value

#### Scenario: Invalid apply_gates_to value
- **WHEN** `provenance.apply_gates_to: sometimes` is configured
- **THEN** `dwarpal check` exits 2 naming `provenance.apply_gates_to` as invalid

#### Scenario: Unknown rule ID in disable_rules
- **WHEN** `gates.ai_patterns.disable_rules: ["no-such-rule"]` is configured
- **THEN** `dwarpal check` exits 2 naming `no-such-rule` as an unknown rule ID

### Requirement: Mode semantics
In `enforce` mode, error-severity findings SHALL block (exit 1). In `warn` mode, findings SHALL be reported but the exit code SHALL be 0. In `ci_strict` mode, error-severity findings SHALL block (exit 1) exactly as in `enforce`, and evidence of a local bypass (e.g. a missing or invalid hook-success marker, per the hook-management contract) on the commits under test SHALL itself be treated as a blocking finding.

#### Scenario: Warn mode never blocks
- **WHEN** a diff exceeds budgets and `mode: warn` is set
- **THEN** findings are printed and the process exits 0

#### Scenario: ci_strict rejects evidence of a local bypass
- **WHEN** `mode: ci_strict` is set and the commit under test lacks a valid hook-success marker (indicating `--no-verify` was used locally)
- **THEN** `dwarpal check` exits 1 with a finding naming the rejected bypass, even though local `enforce` mode does not check for the marker
