## MODIFIED Requirements

### Requirement: Gate contract
Every gate SHALL implement `ID() string` and `Run(ctx, *Diff, RepoIndex) ([]Finding, error)`. Findings SHALL carry `{gate, rule_id, severity, file, line, message, suggestion, docs_url}`. The engine SHALL run enabled gates in configured order, execute independent gates concurrently up to a bounded worker pool, and aggregate all findings.

#### Scenario: Findings carry provenance to their gate
- **WHEN** the diff-budget gate emits a finding
- **THEN** the finding's `gate` field is `diff_budget` and `rule_id` names the specific budget exceeded

#### Scenario: Concurrent execution preserves output determinism
- **WHEN** `dwarpal check` runs with Gate 1, Gate 2, Gate 3, and Gate 4 all enabled against the same staged diff twice in a row
- **THEN** both runs produce byte-identical rendered output (findings sorted by gate registration order, then file, then line) regardless of goroutine scheduling

### Requirement: Report-everything default
With `stop_on_first_block: false` (the default), the engine SHALL run all enabled gates and report all findings even after a blocking finding occurs.

#### Scenario: Multiple violations all reported
- **WHEN** a staged diff exceeds both the line budget and the file budget
- **THEN** the report contains both findings, not just the first

#### Scenario: Findings from multiple gates all reported
- **WHEN** a staged diff exceeds the diff-budget and also adds an out-of-scope file (Gate 4)
- **THEN** the report contains findings from both `diff_budget` and `scope`

### Requirement: Deterministic gates fail closed
If a deterministic gate returns an infrastructure error, the engine SHALL treat the check as blocked (exit 1) and report the gate error — it SHALL NOT silently skip the gate.

#### Scenario: Gate error blocks
- **WHEN** a gate fails with an internal error during `dwarpal check` in enforce mode
- **THEN** the process exits 1 and the report names the failed gate

## ADDED Requirements

### Requirement: Gate ordering and provenance-based filtering
Gates SHALL run in a fixed registry order (cheapest first: diff-budget, branch-policy, ai-patterns, scope). Gates 3 and 4 SHALL be filtered per commit according to `provenance.apply_gates_to` (`agent-only` default runs them only on agent-authored commits; `all-commits` runs them on every commit). Gate 2's branch-policy check is never filtered by `apply_gates_to`.

#### Scenario: Registry order determines default execution order
- **WHEN** `dwarpal rules` lists enabled gates with no custom ordering configured
- **THEN** the order shown is diff-budget, branch-policy, ai-patterns, scope

#### Scenario: apply_gates_to filters ai-patterns and scope, not branch-policy
- **WHEN** `provenance.apply_gates_to: agent-only` and a commit is detected as human-authored and targets a protected branch
- **THEN** the branch-policy gate still evaluates that commit (and reports nothing if the commit is human-authored) while ai-patterns and scope are skipped entirely for it
