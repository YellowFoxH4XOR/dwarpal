# Changelog

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
