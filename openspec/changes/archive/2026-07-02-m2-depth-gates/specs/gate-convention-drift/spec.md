## ADDED Requirements

### Requirement: Convention fingerprint scoring
The drift gate SHALL score added code in the diff against the repo's
convention fingerprint (naming case distribution, import style, error-
handling idiom, file-size norms, per language) sourced from `repo-index`,
and SHALL emit a finding for any added construct that scores as an outlier
above the configured threshold.

#### Scenario: Naming convention outlier
- **WHEN** the repo fingerprint shows 95% of functions use `camelCase` and
  the diff adds a function named `do_the_thing` (snake_case) in the same
  language
- **THEN** the gate emits a finding identifying the naming outlier and the
  repo's dominant convention

#### Scenario: Added code matches repo convention
- **WHEN** all added constructs in the diff score within the repo's normal
  fingerprint distribution
- **THEN** the gate emits no finding

### Requirement: Drift findings default to info severity
Every finding produced by the drift gate SHALL default to `severity: info`
unless a user explicitly overrides `gates.convention_drift.severity` in
config. The gate SHALL NOT change `mode: enforce`'s exit code on its own
when only info-severity findings are present.

#### Scenario: Default config never blocks on drift
- **WHEN** the drift gate fires findings and `gates.convention_drift` has no
  explicit `severity` override
- **THEN** all drift findings carry `severity: info` and `dwarpal check`
  exits 0 if no other gate blocks

#### Scenario: Explicit override raises severity
- **WHEN** `gates.convention_drift.severity: error` is configured
- **THEN** drift findings carry `severity: error` and can cause a block in
  `enforce` mode

### Requirement: Drift gate depends on repo-index availability
The drift gate SHALL depend on `repo-index`'s convention fingerprint being
built; if the fingerprint has not yet been built (e.g. first run, empty
cache), the gate SHALL skip with an informational note rather than blocking
or erroring, and SHALL trigger an index build for subsequent runs.

#### Scenario: No fingerprint on first run
- **WHEN** `dwarpal check` runs in a repo with no `.dwarpal/cache/` index
  yet
- **THEN** the drift gate skips with a note that the fingerprint is being
  built, and does not block
