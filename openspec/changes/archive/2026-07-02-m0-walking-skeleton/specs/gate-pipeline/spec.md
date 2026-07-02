## ADDED Requirements

### Requirement: Gate contract
Every gate SHALL implement `ID() string` and `Run(ctx, *Diff, RepoIndex) ([]Finding, error)`. Findings SHALL carry `{gate, rule_id, severity, file, line, message, suggestion, docs_url}`. The engine SHALL run enabled gates in configured order and aggregate all findings.

#### Scenario: Findings carry provenance to their gate
- **WHEN** the diff-budget gate emits a finding
- **THEN** the finding's `gate` field is `diff_budget` and `rule_id` names the specific budget exceeded

### Requirement: Report-everything default
With `stop_on_first_block: false` (the default), the engine SHALL run all enabled gates and report all findings even after a blocking finding occurs.

#### Scenario: Multiple violations all reported
- **WHEN** a staged diff exceeds both the line budget and the file budget
- **THEN** the report contains both findings, not just the first

### Requirement: Deterministic gates fail closed
If a deterministic gate returns an infrastructure error, the engine SHALL treat the check as blocked (exit 1) and report the gate error — it SHALL NOT silently skip the gate.

#### Scenario: Gate error blocks
- **WHEN** a gate fails with an internal error during `dwarpal check` in enforce mode
- **THEN** the process exits 1 and the report names the failed gate
