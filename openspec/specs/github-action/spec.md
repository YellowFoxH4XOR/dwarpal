# github-action Specification

## Purpose
TBD - created by archiving change m1-deterministic-core. Update Purpose after archive.
## Requirements
### Requirement: Action runs the gate pipeline and uploads SARIF
`uses: dwarpal/action@v1` SHALL run `dwarpal check` against the pull request's diff, produce a SARIF file, and upload it for PR annotation.

#### Scenario: Action annotates a PR with findings
- **WHEN** the Action runs on a PR whose diff violates the diff-budget gate
- **THEN** a SARIF file is produced and passed to the SARIF-upload step, and the Action step's exit status reflects the underlying `dwarpal check` exit code

### Requirement: Action always runs in ci_strict mode
Regardless of the repo's local `.dwarpal.yml` `mode` setting, the Action SHALL force `ci_strict` semantics: evidence of a local bypass (a missing or invalid hook-success marker on the commits under test) SHALL be rejected and treated as a blocking finding.

#### Scenario: Local bypass rejected in Action context
- **WHEN** a commit under test lacks a valid hook-success marker (indicating a local `--no-verify` bypass) and the Action runs against it
- **THEN** the Action reports a blocking finding naming the rejected bypass and exits non-zero

#### Scenario: Repo mode: warn is overridden by Action
- **WHEN** `.dwarpal.yml` sets `mode: warn` and the Action runs against a diff with an error-severity finding
- **THEN** the Action exits non-zero (ci_strict overrides warn-mode's local exit-0 behavior)

### Requirement: Action is a thin wrapper around the compiled binary
The Action SHALL invoke the same `dwarpal` binary used locally (no reimplementation of gate logic in the Action's own runtime), configured via `action/action.yml` inputs mapped to `dwarpal check` flags.

#### Scenario: Action inputs map to CLI flags
- **WHEN** the Action is configured with `config-path: .dwarpal.yml` and `paths: src/**`
- **THEN** the underlying invocation is equivalent to `dwarpal check --paths "src/**"` using that config file, with no separate gate logic executed

