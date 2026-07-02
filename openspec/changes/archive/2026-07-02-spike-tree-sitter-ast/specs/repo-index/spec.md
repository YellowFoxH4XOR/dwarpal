## ADDED Requirements

### Requirement: Incremental rebuild is measured against the p95 < 2s budget
The spike SHALL prototype a `RepoIndex` implementation over a real ~100k-LOC repository and measure the wall-clock time of an *incremental* rebuild — re-indexing only the files changed by a small N-file commit-sized change against a previously built cache — separately from the one-time cold build.

#### Scenario: Cold build measured but not gated
- **WHEN** the prototype builds the index from scratch on the ~100k-LOC repo
- **THEN** the cold build's wall-clock time and peak memory footprint are recorded, but the cold build time alone SHALL NOT be used to decide pass/fail against the p95 < 2s budget

#### Scenario: Incremental rebuild measured against budget
- **WHEN** a commit-sized change (a small number of files, representative of typical diff-budget limits) is applied to the indexed repo and the index is rebuilt incrementally
- **THEN** the incremental rebuild's wall-clock time is recorded and compared against the p95 < 2s pipeline budget (PRD G3)

#### Scenario: Memory footprint recorded
- **WHEN** the incremental rebuild measurement runs
- **THEN** peak memory used by the `RepoIndex` process during the incremental rebuild is recorded alongside the timing result

### Requirement: Go/no-go decision with named fallback
The spike SHALL record an explicit go/no-go decision on the incremental-rebuild approach. If the measured incremental rebuild time exceeds the p95 < 2s budget, the decision record SHALL name a fallback: index sharding by package/directory, or an eager-only build with a documented manual refresh command.

#### Scenario: Incremental rebuild clears the budget
- **WHEN** the measured incremental rebuild time is under the p95 < 2s budget
- **THEN** the decision record states "go" on the incremental-rebuild approach as prototyped

#### Scenario: Incremental rebuild misses the budget
- **WHEN** the measured incremental rebuild time exceeds the p95 < 2s budget
- **THEN** the decision record states "no-go" on the as-prototyped approach and names either index sharding or eager-only-build-with-manual-refresh as the fallback to pursue in M2

### Requirement: RepoIndex skeleton implements the frozen engine interface
If the decision record reaches a "go" (with or without a named fallback applied), the project SHALL land a `RepoIndex` implementation satisfying the `engine.RepoIndex` interface frozen in M0, backed by an on-disk cache under `.dwarpal/cache/`, without changing the interface's signature.

#### Scenario: Existing Gate contract is unaffected
- **WHEN** the `RepoIndex` implementation is landed
- **THEN** the `Gate` interface's `Run(ctx, *Diff, RepoIndex) ([]Finding, error)` signature is unchanged, and gates written against the M0 no-op `RepoIndex` stub continue to compile against the real implementation

#### Scenario: Cache is scoped to a repo-local, git-ignored directory
- **WHEN** the `RepoIndex` implementation builds or rebuilds its cache
- **THEN** cache files are written under `.dwarpal/cache/` within the target repository and are not required to be committed to version control

### Requirement: Index content covers stateful-gate needs measured, not built
The prototype SHALL index at minimum a function inventory (name, file, line range, language) sufficient to support the future duplicate-function and drift gates' data needs, without implementing those gates' matching/scoring logic in this spike.

#### Scenario: Function inventory is queryable
- **WHEN** the prototype's index is queried for a given file
- **THEN** it returns the set of functions defined in that file with name, line range, and language, sourced via the chosen `ast-engine` binding

#### Scenario: Duplicate-function and drift scoring are out of scope
- **WHEN** the spike's `RepoIndex` skeleton is reviewed
- **THEN** it contains no near-duplicate similarity scoring or convention-fingerprint scoring logic — those are explicitly deferred to M1/M2 gate implementation
