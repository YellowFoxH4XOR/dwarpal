## MODIFIED Requirements

### Requirement: Gate ordering and provenance-based filtering
Gates SHALL run in a fixed registry order (cheapest first: diff-budget, branch-policy, ai-patterns, scope). Gates 3 and 4 SHALL be filtered per commit according to `provenance.apply_gates_to` (`all-commits` default runs them on every commit; `agent-only` opt-out runs them only on agent-authored commits). Gate 2's branch-policy check is never filtered by `apply_gates_to`.

#### Scenario: Fixed order
- **WHEN** multiple gates produce findings in one run
- **THEN** findings are reported grouped in registry order regardless of gate execution timing

#### Scenario: Branch policy exempt from filtering
- **WHEN** `provenance.apply_gates_to: agent-only` and a commit is detected as human-authored and targets a protected branch
- **THEN** the branch-policy check still runs (and self-no-ops for the human author)
