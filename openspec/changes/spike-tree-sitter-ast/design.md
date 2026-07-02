## Context

This is a throwaway measurement spike (PRD §6 #2, §8 risk R3, §11 Q3-adjacent), not production work. Two blockers from the M0 feasibility review gate every downstream AST-based gate (Gate 3 AI-patterns, Gate 6 drift, `no-duplicate-function`, user `architecture_rules`):

- **B2 — tree-sitter binding**: the PRD commits to a cgo-free static binary (§5.5, §6 #2) but canonical tree-sitter Go bindings are cgo. WASM grammars on wazero are the candidate cgo-free path but are unproven for parse latency and correctness.
- **B1 — RepoIndex under budget**: stateful gates (drift, duplicate-function) need whole-repo context, which appears to contradict the "diff-first, never whole repo" principle (§6.1) and the p95 < 2s pipeline budget (G3). Whether an *incremental* rebuild (not full rebuild) fits inside that budget on a realistic repo size (~100k LOC) is the open question.

Both blockers are measured against real numbers, not opinion: parse latency, binary size delta, cross-compile success, and incremental-rebuild wall clock on a real ~100k-LOC OSS repo. The output is a decision record plus — only if the numbers support it — a minimal `internal/ast` skeleton behind the existing `engine.RepoIndex` interface (frozen in M0, unchanged by this spike).

## Goals / Non-Goals

**Goals:**
- Produce comparable, reproducible measurements for cgo vs WASM-on-wazero tree-sitter bindings across Go/TS/Python: parse latency per file, resulting binary size, and cross-compile success across darwin/linux/windows × amd64/arm64 with `CGO_ENABLED=0` where applicable.
- Prototype `RepoIndex` cold-build and incremential-rebuild timing and memory footprint on a real ~100k-LOC repo, against the p95 < 2s budget (G3).
- Record an explicit go/no-go decision, with a named fallback, for both B1 and B2.
- If the spike greenlights it, land a minimal `internal/ast` skeleton (parser cache, query runner, language registry) satisfying `ast-engine`, and a real `RepoIndex` implementation satisfying `repo-index` — both behind the interfaces frozen in M0.

**Non-Goals:**
- No production gate logic (Gate 3, Gate 6, `no-duplicate-function`) — those are M1/M2.
- No Rust/Java grammars (M4).
- No changes to the `Gate` interface or `engine.RepoIndex` interface signature — M0 froze these; this spike fills in the implementation only.
- Regex-tier rules are out of scope; they need no AST and proceed independently in M1.

## Decisions

**D1 — Benchmark methodology: same corpus, both bindings, wall-clock + binary size.**
A fixed corpus of real files (subset of the repo used for the RepoIndex prototype, e.g. grafana or kubectl) is parsed with both a cgo tree-sitter binding and a WASM-on-wazero binding, same machine, same files, median of N=20 runs per file after warmup. Binary size measured as the delta a minimal `go build` incurs from embedding each grammar set. Rationale: only an apples-to-apples comparison on the actual PRD-target languages (Go/TS/Python) is decision-grade; synthetic microbenchmarks from upstream tree-sitter are not trusted for this call. Alternative considered: trust published wazero/tree-sitter benchmarks — rejected, none exist for this exact grammar set + Go host combination.

**D2 — Decision rule: cgo vs WASM chosen by measured parse latency + binary size + cross-compile success, in that priority order.**
WASM-on-wazero is selected only if (a) its parse latency is within an agreed tolerance of cgo (documented in the decision record, not pre-committed here — this spike sets the number), AND (b) it cross-compiles cleanly for the full darwin/linux/windows × amd64/arm64 matrix with `CGO_ENABLED=0`. If WASM fails either bar, cgo + goreleaser cross-compile matrix (PRD §8 R3 documented fallback) is the decision, accepting the loss of "cgo-free." Rationale: cgo-free is a goal (§6 #2), not a hard constraint; cross-compile breakage is a hard constraint (every release must ship all target platforms). Alternative: pick WASM regardless of latency, to preserve the cgo-free goal at any performance cost — rejected, would silently violate the p95 < 2s pipeline budget for AST-tier gates.

**D3 — RepoIndex incremental rebuild, not full rebuild, is the thing measured against the 2s budget.**
A full-repo scan happens once at cold start (acceptable to be slow, e.g. on `dwarpal init` or first run) and is cached to `.dwarpal/cache/`. The number that must clear < 2s is the *incremental* rebuild after a small N-file change (matching a typical commit), re-parsing only touched files and patching the cached index. Rationale: this is what runs on every `dwarpal check`; a slow cold build is a one-time cost, a slow incremental rebuild breaks the core UX promise on every commit. Alternative: measure full rebuild against the budget — rejected, sets an unachievably strict bar and doesn't reflect the actual hot path.

**D4 — RepoIndex fallback if incremental rebuild misses budget: sharding or eager-only build.**
Two named fallbacks if D3's number fails: (a) shard the index by package/directory so a small change only rebuilds its shard, or (b) drop incremental rebuild entirely and only build the index eagerly (e.g. as a background daemon or `dwarpal init`-time job), accepting staleness between commits with a documented `dwarpal index refresh` command. Rationale: both are namable, bounded scope changes; open-ended optimization work is out of a spike's scope. Alternative: block M2 entirely until incremental rebuild is fast enough — rejected, PRD ROADMAP already sequences "M2 once repo-index is proven," and sharding/eager fallbacks are real, shippable options.

**D5 — `internal/ast` skeleton is built only if the spike greenlights an approach; it wraps the winning binding behind a stable interface.**
`ast-engine` exposes `Parse(lang, src) (*Tree, error)` and a query runner, hiding whether cgo or wasm is underneath. `repo-index` exposes the frozen `engine.RepoIndex` interface with a real implementation backed by the cache format decided in D3/D4. Rationale: gates 3/6 (M1/M2) code against `ast-engine`/`RepoIndex`, never against the binding directly — keeps a future binding swap (e.g. if WASM matures) a localized change. Alternative: expose the binding's native API directly to gates — rejected, would leak the binding choice into every AST-based gate.

## Risks / Trade-offs

- [Benchmark corpus size/language mix skews the decision] → use the same ~100k-LOC real repo for both the binding benchmark and the RepoIndex prototype so results are consistent and reproducible by a reviewer.
- [WASM grammar embedding size not yet known] → measured directly in D1; if it blows the < 40 MB binary target (§5.5) even with acceptable latency, that alone can flip the decision to cgo.
- [wazero runtime overhead vs native cgo call may dominate on small files, distorting per-file latency for the diff-typical case (few files, few hundred lines)] → benchmark separately on "small diff" (5-20 files) and "whole repo" workloads, since the pipeline's p95 < 2s budget applies to the former.
- [Incremental rebuild correctness (stale entries after renames/deletes) is easy to get wrong under time pressure] → the spike's `RepoIndex` skeleton only needs to be measured for timing, not proven correct for every git operation; correctness hardening is explicitly deferred to M2 gate implementation work, and the decision record must say so.
- [Decision record becomes stale if grammar/library versions move before M1 starts] → decision record pins exact versions of the tree-sitter grammars and bindings evaluated; M1 kickoff re-validates versions haven't moved before relying on the decision.

## Open Questions

- What tolerance (ms delta, or %) between WASM and cgo parse latency counts as "close enough" to prefer WASM's cgo-free benefit? Not fixed in this design — the spike's own measurements should propose a number, since without real data any tolerance chosen now is a guess.
- Which real ~100k-LOC repo(s) to standardize on for both benchmarks (candidate: grafana/kubectl per proposal) — needs a final pick before running the spike, ideally one with representative Go/TS/Python mix if a single repo can't cover all three languages.
- Does `.dwarpal/cache/` need a schema/version field from day one so a future index format change doesn't require every user to blow away their cache silently? Leaning yes, but not required for the spike's measurement goal.
