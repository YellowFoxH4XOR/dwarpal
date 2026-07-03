# Changelog

## v0.1.1

- **Fixed**: `install.sh` strips macOS's `com.apple.quarantine` attribute
  before first run — Gatekeeper SIGKILLs (and removes) unsigned quarantined
  binaries on Apple Silicon
- README rewritten: install paths (Homebrew cask + quarantine note, install
  script, `go install`), the 8-gate table, trust promises
- goreleaser cask commits now authored as `YellowFoxH4XOR
  <yellowfoxh4xor@gmail.com>` instead of the goreleaser bot default

## v0.1.0 (release hardening)

- **Fixed**: pre-push verification no longer blocks merge commits (e.g. GitHub
  PR merges) — a commit with a second parent is treated as verified via its
  parents
- **Fixed**: `dwarpal bypass` is now a functional one-shot override — it arms a
  token the pre-commit hook consumes (gates skipped for exactly one commit,
  push marker still written), on top of the existing audit log + git note
- `dwarpal rules` now reports the duplicate and convention-drift gates
- `dwarpal init` starter config showcases the full gate suite (provenance,
  branch policy, ai_patterns, scope, drift, duplicate; coverage/intent/plugins
  as commented examples)
- goreleaser config migrated off deprecated `brews` to `homebrew_casks`;
  validated with `goreleaser check` + full snapshot cross-compile (6 platforms);
  release workflow wired for a `HOMEBREW_TAP_GITHUB_TOKEN` secret

## M1–M3 — Full gate suite (unreleased)

Deterministic core, depth gates, optional gates, and distribution. AST work is
Go-first via stdlib `go/parser` (spike decision; tree-sitter for TS/Python is
future work — see openspec/ROADMAP.md).

- Gate 2 — provenance detection (env/trailer/branch) + protected-branch policy; `apply_gates_to: agent-only` leaves human commits untouched
- Gate 3 — AI-pattern rules: lint-suppressions, secrets (private key/AWS/assigned), diff-local sql-concat & broad-catch heuristics, and `no-duplicate-function` (token-shingle similarity over the repo function index)
- Gate 4 — scope enforcement + `.dwarpal-task.yml` (`dwarpal task`)
- Gate 5 — diff coverage (lcov/cobertura/go-cover, changed lines, warn-only when absent)
- Gate 6 — convention drift (naming/size, info severity)
- Gate 7 — LLM intent verification (BYO key, fail-open on infra error, off by default)
- Gate 8 — exec plugins (semgrep/gitleaks/etc.)
- Output — SARIF (`check --sarif`) for CI annotation
- CLI — `rules`, `task`, `explain`, `doctor`, `bypass`
- Distribution — goreleaser, Dockerfile, install.sh, GitHub Action, CI/release workflows

## M0 — Walking skeleton (unreleased)

First end-to-end slice: the CLI, config, staged-diff extraction, Gate 1
(diff budget), reporting, and git hooks.

- `dwarpal init` — write starter `.dwarpal.yml` and install bypass-resistant hooks
- `dwarpal check [--json] [--range a..b]` — run the gate pipeline; exit 0/1/2
- `dwarpal hook install|uninstall` — manage hooks (chains to existing hooks)
- Gate 1 — diff budget: max lines/files/new-files with per-glob overrides
- Bypass resistance — pre-commit success marker + pre-push verification catches `--no-verify`
