# Changelog

## Unreleased

- Contribution model: **DCO** (not a CLA) — CONTRIBUTING.md, the DCO text,
  ADR 0002, and a CI workflow enforcing `Signed-off-by` on every PR commit

## Unreleased

- macOS code signing + notarization wired into the release pipeline
  (GoReleaser's built-in quill — cross-platform, no macOS runner). Dormant
  until the Apple secrets are set; see docs/notarization.md for the one-time
  setup. Activating it removes the Gatekeeper `xattr` workaround for users.

## v0.4.1

- **Fixed: `dwarpal check` could hang indefinitely** on large real-world
  repos (observed: 2,167-file TS repo, 30s+ and climbing) — the GLR parser
  ground on real files while the default-on drift gate built the full index
  every check, even with nothing staged. Four layers of fix:
  - empty diffs short-circuit before any index work
  - per-file parse timeout (300ms) + 512KB size cap, heuristic fallback
  - whole-index deadline (5s), heuristic tier past it
  - the index is skipped entirely when the diff touches no indexable language
- **Index disk cache (#67)** (`.dwarpal/cache/`, gob, atomic writes,
  size+mtime invalidation): the 2,167-file repo now checks in **0.86s warm**
  (was 11s every time); drift-only configs (duplicate off) skip function
  extraction entirely — **0.94s cold**, 1.8MB cache

## v0.4.0

- **`dwarpal agent setup <claude-code|codex|opencode|pi>`** — wires the gate
  into the agent's own loop: idempotent managed instruction blocks in
  CLAUDE.md/AGENTS.md (pre-flight workflow, provenance identity, no-bypass
  contract), and for Claude Code a PreToolUse hook merged into
  .claude/settings.json that feeds block JSON straight back to the model
  before the commit attempt
- **`check --range --per-commit`**: evaluate each commit separately —
  budgets are per commit (PRD §5.2), so a range of compliant commits must
  not fail on their sum. The GitHub Action now uses it for PR ranges
  (found when the gate blocked its own maintainer's split PR)

## v0.3.0

- **BREAKING (behavior)**: `provenance.apply_gates_to` now defaults to
  `all-commits` — every commit is gated, regardless of author. Set
  `apply_gates_to: agent-only` to restore the old human-exempt behavior.
  Configs that already set the key are unaffected.

### Documentation

- 25 rule pages (docs/rules/) — every finding's `docs_url` now resolves;
  URLs filled centrally by the engine, `explain` shares the same mapping
  (its hardcoded docs.dwarpal.dev links were dead — unregistered domain)
- Full configuration reference, coverage recipes (7 stacks), integrations
  (GitHub Actions, GitLab CI, pre-commit framework, Docker), and the
  "why harnesses beat prompts" narrative
- `.pre-commit-hooks.yaml` — pre-commit framework consumers can adopt
  Dwarpal without leaving their hooks manager
- README: brew `update`-before-`upgrade` note, documentation section

- `provenance.heuristics`: configurable regex detection signal (4th fallback)
- Override escape: `Dwarpal-Override:` commit trailer (range mode) /
  `DWARPAL_OVERRIDE` env (staged) approves skipping a rule per run
- Drift gains the error-idiom dimension (Go: wrap vs bare vs panic, >=80% rule)
- Engine runs gates concurrently (deterministic output order preserved;
  sequential under stop_on_first_block)
- `dwarpal feedback <rule> --reason`: local-only false-positive log +
  prefilled issue URL — nothing is ever sent automatically

## v0.2.0

### Tree-sitter AST engine (pure Go, CGO-free)

- New `internal/astengine`: tree-sitter parsing + queries for TS/JS/Python via
  the pure-Go gotreesitter runtime — the static-binary promise holds
- `no-duplicate-function` now uses real syntax trees for TS/JS/Python
  (heuristic extractors demoted to parse-failure fallback)
- AST-precise `no-broad-catch` (catch/except body analysis) and
  `no-sql-concat` (template-literal/f-string interpolation) for TS/JS/Python
- Drift gate gains the import-style dimension (Go/TS/JS/Python)
- Index build parallelized across cores + compiled-query cache: 1500-file
  multi-language corpus indexes in 0.7s (was 5.2s); Go stdlib unaffected
- Known limitation (documented): the TS grammar mis-parses typed arrow params;
  tolerant parsing + heuristic supplement covers the gap
- Binary: 7.6MB -> 38MB (206 embedded grammars; under the 40MB PRD cap)

- **architecture_rules** (PRD §5.3): user-defined forbidden-call assertions
  (`matches` regex over go/ast call targets, `forbidden_outside` globs)
- **Entropy secret detection**: Shannon-entropy tier of no-hardcoded-secrets
  (URL/path false positives excluded — found by dogfood)
- **TS/JS + Python duplicate detection**: heuristic function extraction feeds
  the repo index, so no-duplicate-function now covers three languages
- **Anthropic provider** for the intent gate; task manifest id (or branch
  ticket ref like AUTH-42) now feeds the intent prompt
- **Plugin JSON parsing**: gitleaks/semgrep-style output maps to per-finding
  file:line instead of one blob
- **`check --diff <file>`** patch-file mode; **`--explain-for-agent`** alias
- **`stop_on_first_block`** engine option
- **Provenance git notes** (refs/notes/dwarpal-provenance) on passing agent checks
- **Benchmarked**: 140k-LOC repo indexes in ~150ms (13× inside the 2s budget);
  1.8M LOC in 2.4s — incremental caching demoted to >1M-LOC-monorepo work

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
