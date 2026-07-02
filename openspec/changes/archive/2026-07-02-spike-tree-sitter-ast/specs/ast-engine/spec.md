## ADDED Requirements

### Requirement: Binding decision is measured, not assumed
The project SHALL choose between a cgo tree-sitter binding and a WASM-on-wazero tree-sitter binding based on measured parse latency, measured binary size delta, and measured cross-compile success — not on prior assumption. The measurement SHALL cover the v1 AST languages: Go, TypeScript/JavaScript, and Python.

#### Scenario: Same corpus parsed by both bindings
- **WHEN** the spike benchmarks parse latency
- **THEN** both the cgo binding and the WASM-on-wazero binding parse the identical corpus of Go, TypeScript/JavaScript, and Python files on the same machine, and per-file median latency (N=20 runs after warmup) is recorded for each binding

#### Scenario: Binary size delta measured for each binding
- **WHEN** the spike measures binary size
- **THEN** a minimal `go build` embedding the cgo grammars and a separate minimal `go build` embedding the WASM grammars each report their resulting binary size, and the delta between them is recorded

#### Scenario: Cross-compile matrix exercised for each binding
- **WHEN** the spike evaluates cross-compilation
- **THEN** each binding is built for darwin/linux/windows × amd64/arm64, and for each of the 6 targets the build is recorded as succeeding or failing, with `CGO_ENABLED=0` attempted for both bindings

### Requirement: Decision rule prioritizes latency, then size, then cross-compile success
The binding decision SHALL be: prefer WASM-on-wazero only if its measured parse latency is within the tolerance recorded in the decision record AND it cross-compiles successfully on all 6 targets with `CGO_ENABLED=0`; otherwise the decision SHALL be cgo with a goreleaser cross-compile matrix as the documented fallback.

#### Scenario: WASM meets both bars
- **WHEN** WASM-on-wazero's measured latency is within the recorded tolerance of cgo AND all 6 cross-compile targets succeed with `CGO_ENABLED=0`
- **THEN** the decision record names WASM-on-wazero as the chosen binding

#### Scenario: WASM misses either bar
- **WHEN** WASM-on-wazero's measured latency exceeds the recorded tolerance, OR any of the 6 cross-compile targets fails with `CGO_ENABLED=0`
- **THEN** the decision record names cgo as the chosen binding with a goreleaser cross-compile matrix as the fallback delivery mechanism

### Requirement: Decision record is written and reproducible
The spike SHALL produce a written, ADR-style decision record naming the chosen binding, the measured numbers behind the decision, the tolerance used, and the exact grammar/library versions evaluated.

#### Scenario: Decision record names the chosen binding with evidence
- **WHEN** the spike concludes
- **THEN** the decision record states which binding was chosen, cites the measured parse latency, binary size delta, and cross-compile results that justify it, and pins the tree-sitter grammar and binding library versions used

### Requirement: ast-engine skeleton wraps the chosen binding behind a stable interface
If the decision record selects a binding, the project SHALL land a minimal `internal/ast` skeleton exposing a parse cache, a query runner, and a language registry for Go, TypeScript/JavaScript, and Python, without exposing the underlying binding's native API to callers.

#### Scenario: Callers depend on the wrapper, not the binding
- **WHEN** a future gate (outside this spike's scope) needs to parse a file
- **THEN** it calls the `internal/ast` package's parse/query API, and no gate code imports the cgo or wazero binding packages directly

#### Scenario: Skeleton is skipped on no-go
- **WHEN** the decision record concludes neither binding meets a shippable bar
- **THEN** the `internal/ast` skeleton SHALL NOT be landed, and the decision record instead names the blocking gap and what would need to change to unblock it
