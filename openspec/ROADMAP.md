# Dwarpal Roadmap — M0 → M4

The big-picture proposal for all remaining work. Each milestone below is a
separate OpenSpec change (independently reviewable, implementable, archivable);
this document is the connective tissue: sequencing, dependencies, and the
critical path. Milestone framing follows the PRD §10.

## Status snapshot

| Milestone | OpenSpec change | State | Covers |
|---|---|---|---|
| **M0** | `archive/2026-07-02-m0-walking-skeleton` | ✅ shipped | CLI + config + diff extraction + Gate 1 + hooks |
| **Spike** | `spike-tree-sitter-ast` | ✅ **decided (Go-first v1)** | B2 resolved: stdlib `go/parser` for AST (CGO-free); tree-sitter for TS/Python deferred. B1: eager index build, caching is future work |
| **M1** | `m1-deterministic-core` | ✅ **built+tested** | Gate 2 (provenance+branch), Gate 3 (regex + heuristic sql-concat/broad-catch + AST duplicate), Gate 4 (scope+manifest), SARIF, GitHub Action, `rules` |
| **M2** | `m2-depth-gates` | ✅ **built+tested** | Gate 5 (coverage), Gate 6 (drift, Go), `no-duplicate-function` (Go), `explain` |
| **M3** | `m3-launch` | ✅ **built+tested** | Gate 7 (intent, fail-open) + Gate 8 (plugins) + distribution + `doctor`/`bypass` |
| **M4** | *(non-code — this doc)* | planned | Community, TS/Python grammars (tree-sitter), design partners |

**Spike decision (B2), recorded here as the ADR:** v1 uses Go's standard-library
`go/parser` for all AST work (`no-duplicate-function`, convention drift, and the
AST-precise rule tier). Rationale: zero new dependencies, `CGO_ENABLED=0`
preserved, cross-compilation stays trivial — the single-static-binary promise
(§5.5) is worth more at launch than multi-language AST. Cost: v1 AST gates cover
**Go only**; other languages get the regex/heuristic tier. Adding tree-sitter
(wazero-WASM) grammars for TS/Python is a clean future change that slots behind
the same `repoindex`/`RepoIndex` seam without touching gate code.

### Implementation status (what is actually built and tested vs. deferred)

**Built + tested this session (11 packages, race-clean, 18 txtar scenarios):**
- Gate 2 — `internal/provenance` + `internal/gates/branchpolicy` (env/trailer/branch detection; protected-branch blocking; `apply_gates_to: agent-only` filtering so human commits stay untouched)
- Gate 3 — `internal/gates/aipatterns` regex tier: `no-new-lint-suppressions`, 3 secret rules, plus **diff-local v1 heuristics** for `no-sql-concat` and `no-broad-catch` (warn severity, pre-AST)
- Gate 4 — `internal/gates/scope` + `internal/taskmanifest` + `dwarpal task` command
- Gate 5 — `internal/gates/diffcoverage` (lcov/cobertura/go-cover, changed-line coverage, warn-only when artifact missing)
- Gate 7 — `internal/gates/intent` (fail-open on infra error; BYO-key via `DWARPAL_LLM_API_KEY`; off by default)
- Gate 8 — `internal/gates/plugin` (exec contract)
- `internal/report/sarif.go` + `check --sarif`; `dwarpal rules`; `internal/gitio` added-line content enrichment
- Distribution: `.goreleaser.yaml`, `Dockerfile`, `install.sh`, `action/action.yml`, `.github/workflows/{ci,release}.yml` (YAML-valid; `goreleaser check`/`docker build`/Actions-runner verification pending real infra)

**Also built + tested after the spike decision:**
- `internal/repoindex` — Go function inventory (go/parser) + token-shingle similarity + convention fingerprint
- Gate 6 `internal/gates/drift` (naming/size drift, info severity, Go)
- `no-duplicate-function` `internal/gates/duplicate` (Jaccard over shingles, warn, opt-in)
- `dwarpal explain` / `doctor` / `bypass` commands

**Remaining (genuinely future work, documented not faked):**
- TS/Python AST via tree-sitter (v1 is Go-only) — the AST-precise `no-sql-concat`/`no-broad-catch` for non-Go stays on the regex heuristic tier
- Incremental `repoindex` caching under `.dwarpal/cache/` (B1 optimization; v1 builds eagerly, only when a stateful gate is enabled)
- Provenance git-notes attachment; intent gate's task-intent text wired from the manifest
- Real-infra release verification (`goreleaser check`, `docker build`, an Actions run)

## Critical path (what blocks what)

```
M0 ✅
 └─ spike-tree-sitter-ast ──────────────┐  (B1 + B2: gates all AST work)
      │                                 │
      ├─ M1 regex-tier rules, Gate 2,   │   ← can start in parallel with the
      │   Gate 4, SARIF, Action  ───────┤     spike (no AST dependency)
      │                                 │
      └─ M1 AST-tier rules (Gate 3) ◄───┘
           └─ M2 Gate 6 (drift) + no-duplicate-function ◄── needs repo-index
                └─ M3 Gates 7, 8 + distribution + launch
                     └─ M4 listen / community / next grammar
```

**The one hard gate:** `spike-tree-sitter-ast` must close before any AST-based
work (Gate 3 AST tier, Gate 6, duplicate-function). It is the highest-leverage
next step because a bad result invalidates downstream design. Everything with
no AST dependency (Gate 2, Gate 4, regex rules, SARIF, the Action) can proceed
in parallel and de-risks the schedule.

## Blocker → milestone mapping (from the M0 feasibility review)

| Blocker | Resolved by | Notes |
|---|---|---|
| B1 — RepoIndex under 2s budget | `spike-tree-sitter-ast` | Prove incremental rebuild before building stateful gates |
| B2 — tree-sitter cgo vs WASM | `spike-tree-sitter-ast` | cgo + goreleaser matrix is the safe fallback |
| B3 — 5-week timeline optimism | sequencing here | Ship stateless gates first; heuristic gates as `info`/beta |
| B4 — `no-sql-concat` cross-file context | M1 (diff-local v1) → M2 | Ship simple first, add package context once repo-index exists |
| B5 — hook chaining / Windows matrix | M0 ✅ (+ M3 `doctor`) | `dwarpal doctor` reports hook health in M3 |
| B6 — false positives (drift/dup) | M2 | Default `info`; per-rule disable; audited suppression |
| B7 — local bypass culture | M0 ✅ (+ M1 `ci_strict`) | Marker/pre-push in M0; server-side enforcement in M1 |

## Milestone detail

### Spike — `spike-tree-sitter-ast`
Throwaway measurement + decision. Delivers the `ast-engine` and `repo-index`
capabilities only if greenlit. **Exit:** a written cgo-vs-WASM decision and a
proven incremental index rebuild < 2s on a ~100k-LOC repo.

### M1 — `m1-deterministic-core`
The always-on deterministic gates (2, 3, 4), SARIF, and the GitHub Action.
Regex-tier Gate 3 rules + Gates 2/4 have no spike dependency and land first.
**Exit (PRD §10):** all M1 gates dogfooded on Dwarpal's own repo with an agent
as author.

### M2 — `m2-depth-gates`
The artifact/context-dependent gates (5 coverage, 6 drift), the
`no-duplicate-function` rule, and `dwarpal explain`. Heuristic gates ship
advisory. **Exit:** `retry_hints` finalized against real Claude Code/Cursor
loops.

### M3 — `m3-launch`
The optional gates (7 intent BYO-key, 8 exec plugins) and full distribution
(goreleaser, Homebrew, Docker, CI templates) + launch collateral. **Exit:** all
8 gates shipping; Show HN.

### M4 — Listen (non-code, weeks 6–10)
Not an OpenSpec code change — tracked here:
- Triage community-contributed rules/gates; decide CLA vs DCO (§11 Q5).
- Ship the top-requested language grammar (likely Rust or Java).
- Begin Cloud-tier design-partner conversations (persona P3); target ≥3 LOIs
  from 50+ eng companies running `ci_strict` (§9).
- Evaluate the MCP pre-flight server (§11 Q4) for the v1.x roadmap.

## Sequencing recommendation

1. **Now:** `spike-tree-sitter-ast` **and** the no-AST slice of M1 (Gate 2,
   Gate 4, regex rules, SARIF, Action) in parallel.
2. Fold the spike result into M1's AST tier; complete and dogfood M1.
3. M2 once `repo-index` is proven.
4. M3 distribution + launch.
5. M4 listen.

## Scope discipline (what stays out of v1 — PRD §3)

No hosted SaaS/dashboard/org management, no IDE plugins, no code *fixing*, no
non-git VCS, no certain AI-authorship detection. These are open-core Phase 2 or
explicit non-goals — kept out so v1 ships.
