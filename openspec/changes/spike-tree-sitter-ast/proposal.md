## Why

Every AST-based gate (Gate 3 AI-patterns, Gate 6 drift, user `architecture_rules`) depends on two unresolved assumptions the M0 feasibility review flagged as the only true spike blockers:

- **B2 — tree-sitter binding**: canonical Go bindings are cgo; the PRD's cgo-free goal (§5.5, §6 #2, risk R3) requires WASM grammars on a pure-Go runtime (wazero). Unproven.
- **B1 — RepoIndex under budget**: the stateful gates need whole-repo context, contradicting "diff-first, never whole repo" (§6.1). Whether an incremental index rebuild fits the p95 < 2s budget (G3) on a large repo is unproven — and it's the assumption most likely to be wrong.

Writing any AST gate before these are answered risks building on abstractions that a bad spike result invalidates. This change is a **throwaway spike** that produces numbers and a decision, not production code.

## What Changes

- Benchmark **tree-sitter parsing** of Go/TS/Python via (a) cgo bindings vs (b) WASM-on-wazero: parse latency per file, binary size delta, cross-compile matrix (darwin/linux/windows × amd64/arm64 with `CGO_ENABLED=0`).
- Prototype a **`RepoIndex`** over a ~100k-LOC real repo (e.g. grafana/kubectl): cold build time, incremental-rebuild time after an N-file change, and memory footprint — verified against the 2s budget.
- Produce a written **decision record** (ADR-style) choosing cgo vs WASM, and a go/no-go on the incremental-index approach with fallbacks (cgo + goreleaser matrix; index sharding or eager-only build).
- Land a minimal `internal/ast` skeleton (parser cache, query runner, language registry) behind the existing `engine.RepoIndex` interface — only if the spike greenlights it.

## Capabilities

### New Capabilities
- `ast-engine`: tree-sitter wrapper — parse cache, query runner, language registry for Go/TS/Python (PRD §6 `internal/ast`).
- `repo-index`: incremental repo-level index (function inventory, convention fingerprint) built into `.dwarpal/cache/`, consumed by stateful gates (PRD §6.1).

### Modified Capabilities

(none — new subsystems; `gate-pipeline`'s `RepoIndex` stub becomes a real implementation but its interface is unchanged)

## Impact

- New deps depending on the decision: `wazero` + embedded `.wasm` grammars, OR cgo tree-sitter bindings + a goreleaser cross-compile matrix.
- Sets the binary-size and cross-compile story for all releases (§5.5).
- Gates M1's Gate 3 AST-tier rules and M2's Gate 6 + duplicate-function rule. **Nothing downstream should start until this closes.**

## Non-goals

- No production gate logic — this is a measurement/decision spike.
- No Rust/Java grammars (M4).
- Regex-tier rules (which need no AST) are explicitly out of scope here and can proceed in M1 independent of this spike's outcome.
