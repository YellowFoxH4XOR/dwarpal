## ADDED Requirements

### Requirement: Repo function index (Go, eager v1)
`repo-index` SHALL build an in-memory index of the repository's Go functions by walking the work tree and parsing each `.go` file with the stdlib `go/parser`, skipping `.git`, `vendor`, `node_modules`, and `.dwarpal` directories. The index SHALL be built only when at least one index-consuming gate (`no-duplicate-function`, `convention_drift`) is enabled, so runs without stateful gates pay no indexing cost. (Incremental caching under `.dwarpal/cache/` is future work — see ROADMAP blocker B1.)

#### Scenario: Index built only when a consumer gate is enabled
- **WHEN** `dwarpal check` runs with `gates.duplicate.enabled: false` and `gates.convention_drift.enabled: false`
- **THEN** no repo index is built and the pipeline runs with the no-op index

#### Scenario: Broken files skipped
- **WHEN** the repo contains a Go file that fails to parse
- **THEN** that file is skipped and the rest of the repo is still indexed

### Requirement: Function inventory with normalized token shingles
Each indexed function's entry SHALL include its file, name, line range, and a token-shingle set (k-gram token hashes with identifier names and literal values normalized) suitable for `no-duplicate-function` Jaccard-similarity comparison, so near-duplicates survive renames and literal changes.

#### Scenario: Renamed near-duplicate still matches
- **WHEN** two functions are structurally identical but differ in identifier names and literal values
- **THEN** their shingle sets' Jaccard similarity scores at or near 1.0

### Requirement: Convention fingerprint
The index SHALL accumulate a repo convention fingerprint over Go functions — function count, exported count, snake_case-named count, and total function lines (yielding the average function length) — consumable by the drift gate.

#### Scenario: Fingerprint reflects repo style
- **WHEN** the index is built over a repo of predominantly camelCase Go functions
- **THEN** the fingerprint's snake_case ratio is low and its average function length reflects the repo's actual distribution
