## Why

M0 proved the pipeline end-to-end with one gate. M1 delivers the **deterministic core** that makes Dwarpal actually useful: the gates covering agent failure modes 2–5 and 8 (PRD §1), plus the CI-facing outputs (SARIF, GitHub Action) that make rules team-enforceable (G5). This is the PRD's M1 milestone (§10): "all gates dogfooded on Dwarpal's own repo with Claude Code as the authoring agent."

Everything here is deterministic, no-network, fail-closed — extending the M0 contracts, not changing them.

## What Changes

- **Gate 2 — Provenance & Branch Policy** (PRD §5.2): detect agent authorship from `AGENTGATE_AGENT` env, `Co-Authored-By` trailers, branch prefix, heuristic fallback; block agent commits to protected branches; attach provenance as a git note/trailer. Implements `apply_gates_to: agent-only|all-commits`.
- **Gate 3 — AI-Pattern Rules** (PRD §5.2): the built-in rule pack.
  - **Regex tier** (no AST, any language): `no-new-lint-suppressions`, `no-hardcoded-secrets` (entropy + shape) — ship independent of the tree-sitter spike.
  - **AST tier** (Go/TS/Python, needs `spike-tree-sitter-ast`): `no-sql-concat` (diff-local v1 per blocker B4), `no-broad-catch`.
- **Gate 4 — Scope Enforcement** (PRD §5.2): task manifest (`.dwarpal-task.yml` / branch ref / `--paths`), block out-of-scope file changes, always-allow globs; warn-only when no manifest.
- **SARIF output** in `report/` — free GitHub PR annotations (PRD §6 #5).
- **GitHub Action** wrapper (`action/`) + `ci_strict` mode enforcement (bypasses rejected).

## Capabilities

### New Capabilities
- `gate-provenance`: agent detection + branch policy + provenance notes (PRD §5.2 Gate 2).
- `gate-ai-patterns`: rule-pack engine (rules-as-data) + the five v1 rules (PRD §5.2 Gate 3).
- `gate-scope`: task manifest + out-of-scope blocking (PRD §5.2 Gate 4).
- `sarif-output`: SARIF encoder for CI annotation (PRD §6 #5).
- `github-action`: `uses: dwarpal/action@v1` wrapper (PRD §5.5).

### Modified Capabilities
- `gate-pipeline`: add gate ordering/registry for multiple gates, parallel execution, and `apply_gates_to` provenance filtering (M0 defined the interface for one gate; M1 exercises many).
- `config-loading`: extend schema for `provenance`, `gates.ai_patterns`, `gates.branch_policy`, `gates.scope`, `architecture_rules` (M0 validated only diff_budget).
- `cli-core`: add `dwarpal rules` (list active gates/rules) and `dwarpal task` (declare scope manifest).

## Impact

- Depends on `spike-tree-sitter-ast` **only for Gate 3's AST tier**; regex-tier rules, Gates 2 & 4, SARIF, and the Action have no such dependency and can land first.
- New `internal/provenance/`, `internal/gates/{branchpolicy,aipatterns,scope}/`, `rules/` (go:embed), `action/`.
- Establishes the `retry_hints` loop with real Claude Code/Cursor testing.

## Non-goals

- Coverage (Gate 5), drift (Gate 6), duplicate-function, intent (Gate 7), plugins (Gate 8) — M2/M3.
- `no-duplicate-function` (needs RepoIndex) — M2.
- Distribution/goreleaser/Homebrew — M3.
