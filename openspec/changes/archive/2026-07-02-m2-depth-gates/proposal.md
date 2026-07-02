## Why

M1 covers the always-on deterministic gates. M2 adds the **depth gates** that need external artifacts or whole-repo context — the harder, more heuristic ones — plus the agent-facing polish that closes the retry loop. This is the PRD's M2 milestone (§10): Gates 5 (coverage) and 6 (drift, info-only), the duplicate-function rule, the `explain` command, and a finalized `retry_hints` schema tested against real agent loops.

These gates are honest about being heuristic (drift/duplicate default to `info`/advisory), directly mitigating false-positive risk R4.

## What Changes

- **Gate 5 — Diff Coverage** (PRD §5.2): require N% coverage on **changed lines** (default 70%); parse `lcov.info`, Cobertura `coverage.xml`, Go `cover.out`; warn-only when no artifact. Dwarpal consumes artifacts, does not run tests.
- **Gate 6 — Convention Drift** (PRD §5.2): repo convention fingerprint (naming/import/error-handling/file-size distributions) via tree-sitter sampling; score added code, flag outliers. Ships `severity: info` by default.
- **`no-duplicate-function`** rule (PRD §5.2 Gate 3): token-shingle similarity over tree-sitter function nodes vs. the repo function inventory; threshold configurable. The first consumer of `repo-index`.
- **`dwarpal explain <finding-id>`** (PRD §5.1): human-readable rationale + doc link per finding.
- **`retry_hints` schema finalized** (PRD §5.4) against real Claude Code/Cursor loop testing — imperative, machine-consumable remediation.

## Capabilities

### New Capabilities
- `gate-diff-coverage`: changed-line coverage from lcov/cobertura/go-cover (PRD §5.2 Gate 5).
- `gate-convention-drift`: heuristic drift scoring, info-severity (PRD §5.2 Gate 6).
- `explain-command`: `dwarpal explain <id>` rationale lookup (PRD §5.1).

### Modified Capabilities
- `gate-ai-patterns`: add `no-duplicate-function` (consumes `repo-index`).
- `repo-index`: extend fingerprint with convention distributions consumed by drift (spike delivered the function inventory; M2 adds convention stats).
- `cli-core`: add the `explain` command; finalize `retry_hints` in the JSON contract.

## Impact

- **Hard dependency on `spike-tree-sitter-ast`** (`ast-engine` + `repo-index`) — Gates 6 and duplicate-function cannot start until the spike closes and RepoIndex is proven under budget.
- New `internal/gates/{diffcoverage,drift}/`, coverage-format parsers, `docs/` finding rationales.
- Ships copy-paste coverage recipes for the top stacks (mitigates R6).

## Non-goals

- Intent gate (Gate 7), plugins (Gate 8) — M3.
- Running tests to produce coverage — always consumes existing artifacts.
- Making drift/duplicate blocking by default — they stay advisory (R4).
