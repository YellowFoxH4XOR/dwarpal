## 1. Corpus and harness setup

- [ ] 1.1 Pick the ~100k-LOC target repo(s) for both benchmarks (candidate: grafana or kubectl per proposal); vendor or pin a commit SHA for reproducibility
- [ ] 1.2 Build a fixed benchmark corpus of Go/TS/Python files drawn from the target repo(s) for parse-latency measurement
- [ ] 1.3 Set up a benchmark harness (Go `testing.B` or standalone script) that runs N=20 warm iterations per file and records median latency

## 2. B2 — tree-sitter binding spike (cgo vs WASM)

- [ ] 2.1 Wire up a cgo tree-sitter binding (Go/TS/Python grammars) against the benchmark harness
- [ ] 2.2 Wire up a WASM-on-wazero tree-sitter binding (Go/TS/Python grammars) against the same harness
- [ ] 2.3 Run parse-latency benchmark for both bindings on the fixed corpus; separately benchmark "small diff" (5-20 files) vs "whole repo" workloads
- [ ] 2.4 Measure binary size delta: minimal `go build` with cgo grammars embedded vs minimal `go build` with WASM grammars embedded
- [ ] 2.5 Attempt cross-compile for both bindings across darwin/linux/windows × amd64/arm64 with `CGO_ENABLED=0`; record success/failure per target
- [ ] 2.6 Apply the decision rule (design.md D2) to the measured numbers and record the tolerance used
- [ ] 2.7 Write the ADR-style decision record section for B2: chosen binding, measured evidence, pinned grammar/library versions

## 3. B1 — RepoIndex incremental rebuild spike

- [ ] 3.1 Prototype a `RepoIndex` cold-build pass over the target repo(s): function inventory (name, file, line range, language) via the tree-sitter binding chosen (or both, if run before Task 2 concludes)
- [ ] 3.2 Measure cold build wall-clock time and peak memory footprint
- [ ] 3.3 Apply a commit-sized N-file change to the indexed repo; implement and measure incremental rebuild (re-parse only touched files, patch cached index)
- [ ] 3.4 Compare incremental rebuild time against the p95 < 2s pipeline budget (PRD G3); record peak memory during incremental rebuild
- [ ] 3.5 If budget is missed, prototype and re-measure at least one fallback (index sharding by package/directory, or eager-only build) to confirm it is viable before naming it in the decision record
- [ ] 3.6 Write the ADR-style decision record section for B1: go/no-go, measured evidence, fallback (if any)

## 4. Decision record and go/no-go

- [ ] 4.1 Consolidate B1 + B2 decision record sections into a single written ADR-style document (per design.md, referenced from this change)
- [ ] 4.2 Confirm the decision record states an explicit go/no-go for each blocker and, where "no-go" or fallback applies, what M1/M2 must change to proceed (blocked: downstream AST work per proposal Impact)

## 5. ast-engine and repo-index skeleton (only if spike greenlights)

- [ ] 5.1 [blocked on Task 2 decision] If B2 concludes go, land `internal/ast` skeleton: parse cache, query runner, language registry for Go/TS/Python, wrapping the chosen binding only
- [ ] 5.2 [blocked on Task 3 decision] If B1 concludes go (with or without fallback applied), land a `RepoIndex` implementation satisfying the frozen `engine.RepoIndex` interface, backed by `.dwarpal/cache/`
- [ ] 5.3 [blocked on 5.1, 5.2] Verify the M0 `Gate` interface and no-op `RepoIndex` stub still compile unchanged against the new implementation
- [ ] 5.4 [blocked on 5.1, 5.2] Add unit tests for the function-inventory query path (name, file, line range, language) without implementing duplicate-function or drift scoring logic
- [ ] 5.5 If either B1 or B2 concludes no-go without a viable fallback, skip this group entirely and leave the M0 no-op `RepoIndex` stub in place; document the gap in the decision record instead
