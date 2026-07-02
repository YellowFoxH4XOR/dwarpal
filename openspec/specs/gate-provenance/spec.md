# gate-provenance Specification

## Purpose
TBD - created by archiving change m1-deterministic-core. Update Purpose after archive.
## Requirements
### Requirement: Agent-authorship detection order
Dwarpal SHALL detect the authoring agent of a commit by checking, in this fixed order, and stopping at the first match: (1) the `AGENTGATE_AGENT` environment variable, (2) a `Co-Authored-By:` trailer matching a configured agent identity, (3) the branch name against configured `branch_prefixes`, (4) a configurable heuristic fallback (off by default). If no signal matches, the commit SHALL be treated as human-authored.

#### Scenario: Env var wins over trailer
- **WHEN** `AGENTGATE_AGENT=claude-code` is set and the commit also carries a `Co-Authored-By: GitHub Copilot` trailer
- **THEN** provenance is recorded as `claude-code` and the trailer signal is ignored

#### Scenario: Branch prefix used when no env var or trailer present
- **WHEN** a commit on branch `agent/refactor-auth` has no `AGENTGATE_AGENT` env var and no matching trailer
- **THEN** provenance is detected via branch prefix and recorded as agent-authored

#### Scenario: No signal means human-authored
- **WHEN** a commit has no env var, no matching trailer, a non-prefixed branch, and heuristics are disabled
- **THEN** the commit is treated as human-authored and Gate 2's branch-policy check does not block it

### Requirement: Branch policy blocks agent commits to protected branches
Dwarpal SHALL block commits detected as agent-authored from landing directly on any branch matching `protected` (default `["main", "release/*"]`), regardless of `apply_gates_to`.

#### Scenario: Agent commit to main is blocked
- **WHEN** a commit detected as agent-authored targets `main` and `main` is in `protected`
- **THEN** `dwarpal check` exits 1 with a `branch_policy` finding naming the protected branch

#### Scenario: Agent commit to a non-protected branch passes branch policy
- **WHEN** a commit detected as agent-authored targets `agent/feature-x`, which is not in `protected`
- **THEN** the branch-policy check produces no finding

### Requirement: apply_gates_to scopes non-branch-policy gates
`provenance.apply_gates_to` SHALL control which commits Gates 3 and 4 (and future content/context gates) run against: `agent-only` (default) runs them only on commits detected as agent-authored; `all-commits` runs them on every commit. Gate 2's branch-policy check always applies to agent-authored commits regardless of this setting.

#### Scenario: agent-only skips human commits for Gate 3/4
- **WHEN** `apply_gates_to: agent-only` and a commit is detected as human-authored
- **THEN** Gate 3 and Gate 4 report no findings for that commit, even if it would otherwise violate a rule

#### Scenario: all-commits runs Gate 3/4 on human commits too
- **WHEN** `apply_gates_to: all-commits` and a commit is detected as human-authored
- **THEN** Gate 3 and Gate 4 run and may report findings for that commit

