# sarif-output Specification

## Purpose
TBD - created by archiving change m1-deterministic-core. Update Purpose after archive.
## Requirements
### Requirement: SARIF encoding of findings
`dwarpal check --sarif <path>` SHALL write a SARIF 2.1.0 document to the given path derived from the same `[]Finding` model used by the `tty` and `json` renderers, with no changes to the `Finding` schema.

#### Scenario: SARIF written on block
- **WHEN** `dwarpal check --sarif results.sarif` runs against a diff with one finding
- **THEN** `results.sarif` is written containing a SARIF `run` with one `result` entry, and the process exit code follows the normal 0/1/2 contract

### Requirement: Severity maps to SARIF level
Dwarpal SHALL map `Finding.severity` to SARIF `level`: `error` → `error`, `warn` → `warning`, `info` → `note`.

#### Scenario: Error severity maps to SARIF error level
- **WHEN** a `diff_budget` finding with `severity: error` is encoded to SARIF
- **THEN** the corresponding SARIF result's `level` is `error`

#### Scenario: Info severity maps to SARIF note level
- **WHEN** a `convention_drift`-style finding with `severity: info` is encoded to SARIF
- **THEN** the corresponding SARIF result's `level` is `note`

### Requirement: Rule metadata and location included
Each SARIF result SHALL carry `ruleId` set to `Finding.rule_id`, a `message.text` set to `Finding.message`, a `helpUri` set to `Finding.docs_url` when present, and a physical location referencing `Finding.file` and `Finding.line`.

#### Scenario: SARIF result carries file and line
- **WHEN** a finding with `file: "internal/auth/login.go", line: 42` is encoded to SARIF
- **THEN** the SARIF result's physical location `artifactLocation.uri` is `internal/auth/login.go` and `region.startLine` is `42`

### Requirement: SARIF can be combined with other output modes
`--sarif` SHALL be combinable with `--json` or the default TTY output in the same invocation; SARIF is written to its file path while stdout follows the `--json` or TTY contract unaffected.

#### Scenario: SARIF and JSON together
- **WHEN** `dwarpal check --json --sarif results.sarif` runs
- **THEN** stdout contains only the JSON document per the cli-core JSON contract, and `results.sarif` is additionally written to disk

