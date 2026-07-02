## MODIFIED Requirements

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
If a deterministic gate returns an infrastructure error, the engine SHALL treat the check as blocked (exit 1) and report the gate error — it SHALL NOT silently skip the gate. The sole exception is the intent-verification gate (Gate 7): an infrastructure failure of that gate specifically (provider timeout, network error, auth error, malformed response) SHALL NOT by itself cause a block; it SHALL be reported as a `warn`-severity finding. A negative verdict successfully returned by the intent gate is not an infrastructure error and follows the normal blocking rule. Plugin gates (Gate 8) and all other gates remain fail-closed with no exception.

#### Scenario: Gate error blocks
- **WHEN** a gate fails with an internal error during `dwarpal check` in enforce mode
- **THEN** the process exits 1 and the report names the failed gate

#### Scenario: Intent gate infra error does not block
- **WHEN** the intent-verification gate fails due to a provider timeout during `dwarpal check` in enforce mode
- **THEN** the process does not exit 1 solely because of that failure, and the report includes a `warn`-severity finding naming the infra failure

#### Scenario: Plugin gate error still blocks
- **WHEN** a configured plugin gate fails to execute (e.g., missing binary) during `dwarpal check` in enforce mode
- **THEN** the process exits 1 and the report names the failed plugin gate
