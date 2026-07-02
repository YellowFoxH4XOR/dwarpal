## MODIFIED Requirements

### Requirement: Incremental repo-level index
`repo-index` SHALL maintain a repo-wide index in `.dwarpal/cache/`, built
once (cold build) and thereafter rebuilt incrementally on only the files
changed since the last build, keeping stateful-gate invocations within the
p95 < 2s pipeline budget. The index SHALL include, per language: a function
inventory (name, location, token-shingle set) and a convention fingerprint
(naming case distribution, import style distribution, error-handling idiom
distribution, file-size distribution).

#### Scenario: Cold build
- **WHEN** `dwarpal check` runs in a repo with no existing
  `.dwarpal/cache/` index
- **THEN** the index is built from the full repo tree before gates that
  depend on it run

#### Scenario: Incremental rebuild after N-file change
- **WHEN** an existing index is present and only a small subset of files
  changed since the last build
- **THEN** only those files are re-parsed and re-indexed; unaffected
  entries are reused from cache

#### Scenario: Function inventory includes shingle sets
- **WHEN** the index is built or incrementally updated for a file
  containing function definitions
- **THEN** each function's entry includes a token-shingle set suitable for
  `no-duplicate-function` similarity comparison

#### Scenario: Convention fingerprint includes all four dimensions
- **WHEN** the index is built or incrementally updated for a given language
- **THEN** the language's fingerprint includes naming-case, import-style,
  error-handling-idiom, and file-size distributions, each consumable by the
  drift gate

#### Scenario: Index build stays within budget
- **WHEN** an incremental rebuild runs after a typical single-commit change
  (a handful of files)
- **THEN** the rebuild completes fast enough that the overall `dwarpal
  check` p95 remains under the 2s budget on the reference ~100k-LOC repo
