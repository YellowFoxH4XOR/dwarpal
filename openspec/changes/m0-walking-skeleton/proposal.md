## Why

Dwarpal exists only as a PRD (dwarpal-prd.md); no code exists yet. PRD §10 M0 defines the exit criterion: `dwarpal init && dwarpal check` blocks an oversized staged diff in under 1 second. Building this walking skeleton first forces every cross-cutting contract (Gate interface, Finding schema, output formats, exit codes, hook install) to exist end-to-end before any expensive gate work, so later gates slot into proven abstractions instead of speculative ones.

## What Changes

- Scaffold the Go module and repo layout per PRD §6 (`cmd/dwarpal`, `internal/{config,gitio,engine,gates,report,hooks}`).
- Cobra CLI with `init`, `check` (incl. `--json`), `hook install|uninstall`, `version` commands.
- Config loading/validation for a minimal `.dwarpal.yml` (koanf): `version`, `mode`, `gates.diff_budget`.
- Staged-diff extraction via shell-out to system `git` (`internal/gitio`): changed files, added/removed line counts, hunks.
- Engine that runs gates in order and aggregates findings (PRD §5.2 pipeline, single gate for now).
- **Gate 1 — Diff Budget**: max changed lines / max files / max new files with per-path-glob overrides (PRD §5.2 Gate 1).
- Report layer: TTY renderer and JSON encoder over one findings model; exit codes 0/1/2 as contract (PRD §5.4).
- Hook install via `core.hooksPath` with chaining to pre-existing hooks, plus bypass resistance: hook-success marker file checked by a pre-push hook (PRD §8 R7).
- testscript/txtar test harness wired up with fixture repos.

## Capabilities

### New Capabilities
- `cli-core`: command surface, exit-code contract, `--json` output mode (PRD §5.1, §5.4).
- `config-loading`: `.dwarpal.yml` discovery, parse, validate, defaults (PRD §5.3).
- `diff-extraction`: staged-diff model (files, hunks, line counts) from system git (PRD §6).
- `gate-pipeline`: engine ordering, finding aggregation, fail-closed semantics for deterministic gates (PRD §5.2, §6).
- `gate-diff-budget`: line/file/new-file budgets with per-glob overrides (PRD §5.2 Gate 1).
- `hook-management`: install/uninstall via core.hooksPath, chaining, bypass-resistant marker + pre-push check (PRD §5.1, §8 R7).

### Modified Capabilities

(none — greenfield)

## Impact

- New Go module `github.com/YellowFoxH4XOR/dwarpal` (module path TBD-confirm; PRD reserves the `dwarpal` GitHub org).
- New dependencies: spf13/cobra, knadh/koanf, rogpeppe/go-internal (test-only).
- Requires system `git` at runtime (explicit design choice — gh-CLI style shell-out).
- No tree-sitter/AST work in this change (deferred to the spike + M1); no network calls, no telemetry.

## Non-goals

- Gates 2–8 (provenance, AST rules, scope, coverage, drift, intent, plugins) — later changes.
- SARIF output (M1), `explain`/`rules`/`bypass`/`doctor` commands (M1+).
- tree-sitter spike (wazero vs cgo) and RepoIndex latency benchmark — separate spike change, can run in parallel.
- Homebrew/goreleaser/GitHub Action distribution (M3).
- Windows hook test matrix (PRD §11 Q3) — tracked, not blocking M0.
