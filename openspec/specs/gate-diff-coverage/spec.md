# gate-diff-coverage Specification

## Purpose
TBD - created by archiving change m2-depth-gates. Update Purpose after archive.
## Requirements
### Requirement: Changed-line coverage threshold
The diff-coverage gate SHALL compute a coverage percentage over only the
diff's added/modified lines and SHALL emit a blocking finding when that
percentage is below the configured `gates.diff_coverage.min_percent`
(default 70). Untouched lines in a changed file SHALL NOT count toward the
percentage.

#### Scenario: Below threshold blocks
- **WHEN** a staged diff adds 100 lines, a coverage artifact shows 60 of
  those lines covered, and `min_percent` is 70
- **THEN** the gate emits a finding with `severity: error` naming the
  actual (60%) and required (70%) percentages

#### Scenario: At or above threshold passes
- **WHEN** a staged diff's added lines are covered at or above
  `min_percent` per the artifact
- **THEN** the gate emits no finding

#### Scenario: Only added lines count
- **WHEN** a changed file has 500 pre-existing uncovered lines and the diff
  adds 10 fully covered lines
- **THEN** the gate computes 100% coverage for that file's contribution,
  ignoring the 500 untouched lines

### Requirement: Coverage artifact format parsing
The gate SHALL parse `lcov.info`, Cobertura `coverage.xml`, and Go
`cover.out` formats into a common per-file, per-line covered/uncovered
model, auto-detecting format from file content rather than extension.

#### Scenario: lcov artifact
- **WHEN** the configured artifact is a valid `lcov.info` file with `SF:`
  and `DA:` records
- **THEN** the gate parses per-line coverage for each `SF:` file

#### Scenario: Cobertura artifact
- **WHEN** the configured artifact is a valid Cobertura `coverage.xml` file
- **THEN** the gate parses per-line `hits` counts into covered/uncovered

#### Scenario: Go cover.out artifact
- **WHEN** the configured artifact is a Go `cover.out` file starting with a
  `mode:` line
- **THEN** the gate parses each covered block's line range and count

#### Scenario: Unrecognized format is a gate error
- **WHEN** the configured artifact exists but matches none of the three
  supported formats
- **THEN** the gate returns an infrastructure error and the pipeline fails
  closed per gate-pipeline's existing error-handling rule

### Requirement: Missing artifact is warn-only
The gate SHALL treat a missing coverage artifact as a non-blocking,
skippable condition: when no artifact is found at the configured path or
any auto-detected default location, it SHALL emit no blocking finding and
SHALL report that coverage was skipped.

#### Scenario: No artifact configured or present
- **WHEN** `gates.diff_coverage` is enabled but no artifact exists at the
  configured path and none of the default filenames are present
- **THEN** `dwarpal check` proceeds without a coverage finding and the
  human/JSON report notes coverage was skipped

### Requirement: Stale artifact warning
The gate SHALL compare the coverage artifact's modification time against
the diff's base commit time and SHALL emit an advisory (non-blocking)
finding when the artifact predates the diff, since coverage from before the
agent's edits cannot be trusted for those lines.

#### Scenario: Artifact older than diff base
- **WHEN** the coverage artifact's mtime is earlier than the diff's base
  commit timestamp
- **THEN** the gate emits a `severity: info` finding noting the artifact
  may be stale, in addition to (or instead of, if unparseable) the coverage
  check

