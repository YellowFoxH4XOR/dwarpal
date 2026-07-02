## Context

M0 shipped the walking skeleton: CLI → config → staged diff → one gate
(diff-budget) → report → exit code → hooks. The `Gate` interface, `Finding`
schema, exit-code contract, and JSON output shape are frozen (M0 design D2–D3)
and MUST NOT change shape in M1 — this change only adds gates that implement
the existing contract, plus outputs that consume the existing `Finding` model.

M1 delivers Gates 2, 3, 4 (PRD §5.2), SARIF output (PRD §5.4/§6 #5), and the
GitHub Action (PRD §5.5). It is the PRD's largest milestone and the one that
makes Dwarpal "actually useful" per the M1 exit criterion (PRD §10): all M1
gates dogfooded on Dwarpal's own repo with an agent as author.

The hard external constraint: `spike-tree-sitter-ast` (B1 RepoIndex latency,
B2 cgo-vs-WASM) is **not yet resolved**. Per ROADMAP.md, only Gate 3's AST
tier (`no-sql-concat`, `no-broad-catch`) depends on it. Gate 2, Gate 4,
Gate 3's regex tier, SARIF, and the Action have zero AST dependency and are
designed here to be implementable and shippable independently of the spike
outcome.

## Goals / Non-Goals

**Goals:**
- Extend the M0 engine from one gate to many: ordered registry, parallel-safe
  execution, `apply_gates_to` provenance filtering (PRD §5.2 Gate 2).
- Ship Gate 2 (provenance + branch policy), Gate 4 (scope), and Gate 3's
  regex tier without waiting on the tree-sitter spike.
- Design Gate 3's AST tier so it's a drop-in addition once the spike lands —
  same rule-pack contract, just a different matcher backend.
- SARIF encoder reusing the existing `Finding` model (no model changes) —
  M0 design D3 anticipated this.
- GitHub Action as a thin wrapper: pull binary (or use container), run
  `dwarpal check --json`/`--sarif`, upload SARIF, honor `ci_strict`.
- `ci_strict` mode: bypasses (`dwarpal bypass`) rejected in this mode —
  closes blocker B7 (local bypass culture) at the CI boundary.

**Non-Goals:**
- Gate 3's AST tier implementation is speculative here (interfaces + regex
  substitutes only) until the spike closes; tasks.md flags exactly which
  tasks are blocked.
- Gate 5 (coverage), Gate 6 (drift), `no-duplicate-function` (needs
  RepoIndex) — M2.
- Distribution (goreleaser, Homebrew) — M3; the Action in M1 assumes a
  locally built or `go install`ed binary / a minimal Dockerfile, not a
  polished release pipeline.
- Server-side bypass *rejection infrastructure* beyond "the Action checks
  for and rejects a bypass marker when `ci_strict: true`" — no hosted
  service, no webhook receiver (that's Cloud/Phase 2, PRD §7).

## Decisions

**D1 — Provenance detection order is fixed and short-circuiting.**
`internal/provenance` checks, in order: `AGENTGATE_AGENT` env var →
`Co-Authored-By:` trailer matching configured agent identities → branch
prefix match → heuristic fallback (commit message patterns, off by default).
First match wins and is recorded; ties are impossible by construction.
Rationale: PRD §5.2 specifies this exact order; determinism matters because
provenance drives which gates even run (`apply_gates_to: agent-only`).
Alternative considered: score all signals and take the highest-confidence —
rejected, adds nondeterminism and a tunable threshold for no proven benefit
at v1 scale.

**D2 — Provenance is attached via git notes, not commit amendment.**
`git notes --ref=dwarpal-provenance add` records `{agent, detected_via,
timestamp}` post-commit; commits are never rewritten. Rationale: matches PRD
§5.2 ("attach provenance as a git note/trailer so git blame forensics work
later") and avoids the SHA-churn and force-push implications of amending.
Alternative: append a trailer via `commit --amend` — rejected, mutates
history the agent already produced and breaks hash-based dedup elsewhere
(e.g., the M0 hook-success marker is keyed to tree hash, not commit SHA, so
this is safe, but amending commits is not).

**D3 — Rule pack is data (YAML + Go matcher registry), not one Go file per
rule.**
Each built-in rule is `{id, description, severity, languages, tier: regex|ast,
matcher}`. Regex-tier matchers are compiled `regexp.Regexp` + line-added
filtering (only flag lines the diff adds, never pre-existing code — this is
what "AI-pattern" means: catching what the agent just introduced). AST-tier
matchers are tree-sitter queries (deferred to spike). The registry is
embedded via `go:embed rules/*.yml` per PRD §6. Rationale: matches
`gitleaks`/`semgrep` rules-as-data precedent named in config.yaml exemplars;
lets the community add rules without touching `internal/engine`.
Alternative: hardcode each rule as a Go function implementing a shared
interface — rejected for regex-tier (no expressiveness benefit, harder to
contribute to), but AST-tier rules with cross-line context may still need a
thin Go shim around the query — acceptable hybrid, decided at spike time.

**D4 — `no-sql-concat` and `no-broad-catch` ship as diff-local v1 (PRD
blocker B4).**
AST tier operates only on the changed file's added lines plus same-file
context (no cross-package resolution). Rationale: ROADMAP.md explicitly
sequences full package-context `no-sql-concat` to M2 once RepoIndex exists;
shipping diff-local first avoids blocking M1 on B1's resolution.

**D5 — Scope enforcement resolves the task manifest with a fixed precedence:
`--paths` flag > `.dwarpal-task.yml` on branch > parsed ticket ref in
branch/commit message > none (warn-only).**
Rationale: PRD §5.2 lists these three manifest sources; an explicit
precedence avoids ambiguity when more than one is present (e.g., a CLI flag
should always win over a stale committed manifest). `always_allow` globs
(lockfiles, snapshots) apply regardless of source.

**D6 — SARIF encoder is a pure function over `[]Finding`, third printer
alongside `tty`/`json` (M0 design D3's anticipated slot).**
`report.Render(w, findings, report.SARIF)` maps `Finding.severity` →
SARIF `level` (error→error, warn→warning, info→note), `rule_id` → SARIF
`ruleId`, `docs_url` → `helpUri`. No new Finding fields required. Rationale:
zero-cost extension point M0 designed for; keeps SARIF from becoming a
parallel model that drifts from `tty`/`json`.

**D7 — GitHub Action is a composite/Docker action invoking the built binary,
not a reimplementation in JS/TS.**
`action/action.yml` (Docker-based) runs `dwarpal check --sarif=results.sarif
--json` inside the container, uploads SARIF via
`github/codeql-action/upload-sarif`, and sets `ci_strict: true` implicitly
(Action runs are never local — bypass rejection is unconditional in Action
context). Rationale: keeps the entire product logic in Go; the Action is
glue, consistent with `config.yaml`'s "single static binary" tech decision.
Alternative: a JS action re-implementing gate calls — rejected, duplicate
logic in two languages is exactly the drift M1 exists to prevent elsewhere.

**D8 — Engine goes from sequential (M0 D4) to bounded parallel, order still
deterministic in output.**
Gates run concurrently (bounded worker pool, default = NumCPU) but findings
are sorted into a stable order (gate registration order, then file, then
line) before rendering, so `report-everything` output is reproducible
byte-for-byte across runs. Rationale: PRD's p95 < 2s budget on larger diffs
with 3+ gates now active; concurrency is required to hold that budget, but
the M0 spec's "deterministic" framing must survive — determinism is about
*output*, not *execution order*. Alternative: keep sequential — rejected,
risks the latency budget once Gate 3 (regex tier over every added line) and
Gate 4 (manifest resolution) run alongside Gate 1/2.

## Risks / Trade-offs

- [Provenance heuristic fallback is inherently uncertain (PRD is explicit:
  no certain AI-authorship detection is a v1 non-goal)] → heuristic tier is
  opt-in (`off` by default), and its findings are always labeled
  `detected_via: heuristic` in the provenance note so downstream tooling can
  discount confidence.
- [Regex-tier secret detection (entropy + shape) will false-positive on
  high-entropy non-secrets (hashes, generated IDs)] → ship as `error`
  severity only for known key-shape prefixes (e.g., `sk-`, `AKIA`); pure
  entropy matches ship as `warn` by default, tunable via
  `disable_rules`/threshold config.
- [Concurrent gate execution (D8) introduces a class of bugs (races over
  shared RepoIndex stub) not present in M0's single-gate engine] →
  `RepoIndex` remains a read-only interface in M1 (still a no-op/stub per M0
  D2); no gate mutates shared state, so no locking is needed yet — revisit
  when M2 makes RepoIndex real.
- [AST-tier tasks are blocked on an external spike with no fixed close date]
  → tasks.md sequences all non-AST work first and marks AST tasks
  `BLOCKED: spike-tree-sitter-ast`; M1 can archive/ship its non-AST slice
  independently if the spike overruns (ROADMAP.md sequencing).
- [GitHub Action tied to a Docker image adds a release artifact not yet
  covered by goreleaser (that's M3)] → M1 ships a hand-built minimal
  Dockerfile sufficient for the Action to dogfood on Dwarpal's own repo (PRD
  §10 exit criterion); production multi-arch image publishing is explicitly
  M3 scope (proposal Non-goals).

## Open Questions

- Exact entropy threshold and key-shape regex list for `no-hardcoded-secrets`
  — needs tuning against a small corpus of real leaked-key examples before
  default severity is finalized; tracked as a task, not blocking spec
  approval (thresholds are config, not contract).
- Should `apply_gates_to: all-commits` re-run Gate 2 itself (branch-policy
  check) even for human commits, or does it only affect *which other gates*
  provenance-gates itself? Reading PRD §5.2 literally, Gate 2's branch-policy
  half always applies to agent-flagged commits regardless of this setting;
  `apply_gates_to` governs Gates 3/4 (and later 5/6). Spec below encodes this
  reading — flag for owner confirmation during review.
