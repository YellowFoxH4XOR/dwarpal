# hook-management Specification

## Purpose
TBD - created by archiving change m0-walking-skeleton. Update Purpose after archive.
## Requirements
### Requirement: Hook install via core.hooksPath with chaining
`dwarpal hook install` SHALL set `core.hooksPath` to a dwarpal-managed directory containing `pre-commit` and `pre-push` scripts. If `core.hooksPath` was already set or hooks exist in `.git/hooks`, the displaced hooks SHALL be recorded and invoked by the dwarpal hooks before dwarpal's own logic. If chaining is impossible, install SHALL refuse with a clear message rather than clobber.

#### Scenario: Clean install
- **WHEN** `dwarpal hook install` runs in a repo with no existing hooks
- **THEN** core.hooksPath points to the dwarpal hooks directory and both scripts exist and are executable

#### Scenario: Coexistence with existing hooks
- **WHEN** the repo already has a `.git/hooks/pre-commit` (e.g., husky)
- **THEN** a commit triggers the pre-existing hook first, then `dwarpal check`, and both must pass

#### Scenario: Uninstall restores prior state
- **WHEN** `dwarpal hook uninstall` runs after an install that displaced a prior hooksPath
- **THEN** the prior core.hooksPath value is restored

### Requirement: Bypass-resistant marker and pre-push verification
On a passing pre-commit check, dwarpal SHALL write a marker keyed to the staged tree hash inside `.git/`. The dwarpal pre-push hook SHALL refuse to push commits whose tree lacks a valid marker, so commits created with `git commit --no-verify` are caught at push time. The refusal message SHALL name the offending commits and the re-check command.

#### Scenario: no-verify commit caught at push
- **WHEN** a commit is created with `--no-verify` (skipping pre-commit) and then pushed
- **THEN** the pre-push hook blocks the push and names the unverified commit

#### Scenario: Legitimate flow passes
- **WHEN** a commit passes the pre-commit check and is then pushed
- **THEN** the pre-push hook finds a valid marker and allows the push

### Requirement: Hooks never trap the user on infrastructure failure
If the `dwarpal` binary is missing at hook time, hooks SHALL print how to fix or uninstall (`dwarpal hook uninstall` / `git config --unset core.hooksPath`) rather than failing with an inscrutable error.

#### Scenario: Binary removed after install
- **WHEN** a commit is attempted after the dwarpal binary was deleted from PATH
- **THEN** the hook's error message includes the uninstall instruction

