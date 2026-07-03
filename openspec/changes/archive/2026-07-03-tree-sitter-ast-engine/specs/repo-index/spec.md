## MODIFIED Requirements

### Requirement: Repo function index (Go, eager v1)
`repo-index` SHALL build an in-memory index of the repository's functions by walking the work tree, skipping `.git`, `vendor`, `node_modules`, and `.dwarpal` directories. Go files SHALL be parsed with the stdlib `go/parser`; TypeScript/JavaScript and Python files SHALL be parsed via the `ast-engine` tree-sitter runtime, with the v1 heuristic extractors retained as automatic fallback when AST parsing fails for a file. The index SHALL be built only when at least one index-consuming gate (`no-duplicate-function`, `convention_drift`) is enabled, so runs without stateful gates pay no indexing cost.

#### Scenario: Index built only when a consumer gate is enabled
- **WHEN** `dwarpal check` runs with `gates.duplicate.enabled: false` and `gates.convention_drift.enabled: false`
- **THEN** no repo index is built and the pipeline runs with the no-op index

#### Scenario: Broken files skipped
- **WHEN** the repo contains a Go file that fails to parse
- **THEN** that file is skipped and the rest of the repo is still indexed

#### Scenario: TS function extracted via tree-sitter
- **WHEN** the repo contains a TypeScript file with a class method
- **THEN** the index contains that method with accurate start/end lines from the syntax tree

#### Scenario: AST failure falls back to heuristics
- **WHEN** a Python file fails tree-sitter parsing
- **THEN** the heuristic indent-based extractor indexes it instead, and the run continues

## ADDED Requirements

### Requirement: Import-style fingerprint dimension
The convention fingerprint SHALL record per-language import-form distributions (Go: grouped vs single imports; TS/JS: named vs default vs namespace vs require; Python: `import` vs `from ... import`), computed from import nodes during index build, consumable by the drift gate.

#### Scenario: Fingerprint captures import forms
- **WHEN** the index is built over a repo whose TS files overwhelmingly use named imports
- **THEN** the fingerprint's TS import distribution shows named imports as the dominant form
