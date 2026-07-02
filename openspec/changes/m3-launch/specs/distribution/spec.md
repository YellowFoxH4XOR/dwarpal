## ADDED Requirements

### Requirement: goreleaser-driven release matrix
Releases SHALL be produced by a single `.goreleaser.yaml` covering darwin/linux/windows on amd64/arm64, publishing static binaries (`CGO_ENABLED=0` where the resolved tree-sitter binding strategy allows), checksums, and an SBOM, attached to a GitHub Release on each tag push.

#### Scenario: Tagged release produces the full matrix
- **WHEN** a semver tag is pushed to the repository
- **THEN** the GitHub Release for that tag contains archives for darwin/linux/windows across amd64/arm64 plus a checksums file

### Requirement: Homebrew tap formula published per release
Each goreleaser run SHALL update a Homebrew tap formula so that `brew install dwarpal/tap/dwarpal` installs the version matching the release tag.

#### Scenario: Homebrew install matches the release
- **WHEN** a user runs `brew install dwarpal/tap/dwarpal` after a release
- **THEN** the installed `dwarpal version` output matches the released tag

### Requirement: Docker image published per release
A `scratch`-based, multi-arch Docker image SHALL be published per release, suitable for CI use, containing only the static binary and required runtime assets (embedded grammars).

#### Scenario: Docker image runs check
- **WHEN** `docker run dwarpal/dwarpal:<tag> check --json` is invoked against a mounted repo
- **THEN** it produces the same JSON output shape as the native binary

### Requirement: Installable in under 60 seconds
At least one supported install path (Homebrew, `go install`, curl script, or Docker) SHALL take a user from "nothing installed" to a working `dwarpal version` in under 60 seconds on a typical broadband connection, satisfying PRD goal G2.

#### Scenario: curl script quick install
- **WHEN** a user runs the published curl install script on a supported platform
- **THEN** `dwarpal version` succeeds within 60 seconds of starting the script

### Requirement: CI templates ship for GitHub Action and GitLab CI
Dwarpal SHALL ship a GitHub Action wrapper (`uses: dwarpal/action@v1`) invoking `dwarpal check --json` and annotating via SARIF, and a GitLab CI template, and a pre-commit-framework hook definition, each documented with a copy-paste example.

#### Scenario: GitHub Action annotates a PR
- **WHEN** the GitHub Action runs on a pull request with a blocking finding
- **THEN** the workflow fails and the finding appears as a PR annotation via the emitted SARIF

### Requirement: No telemetry, no unconfigured network calls
None of the distributed artifacts (binary, Docker image, Action) SHALL make any network call during normal operation except the intent-verification gate (when explicitly configured) or an `exec` plugin the user configured. No artifact SHALL contain telemetry or analytics code.

#### Scenario: Default operation is network-silent
- **WHEN** `dwarpal check` runs with default configuration (no `intent_check`, no `plugins`) on any distributed artifact
- **THEN** no outbound network connection is made
