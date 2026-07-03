## MODIFIED Requirements

### Requirement: apply_gates_to scopes non-branch-policy gates
`provenance.apply_gates_to` SHALL control which commits Gates 3 and 4 (and future content/context gates) run against: `all-commits` (default) runs them on every commit; `agent-only` is the explicit opt-out that runs them only on commits detected as agent-authored. Gate 2's branch-policy check always applies to agent-authored commits regardless of this setting.

#### Scenario: all-commits default gates human commits
- **WHEN** no `apply_gates_to` is configured and a human-authored commit exceeds a budget
- **THEN** the content gates run and the commit is blocked

#### Scenario: agent-only opt-out skips human commits for Gate 3/4
- **WHEN** `apply_gates_to: agent-only` and a commit is detected as human-authored
- **THEN** Gates 3 and 4 do not run for that commit
