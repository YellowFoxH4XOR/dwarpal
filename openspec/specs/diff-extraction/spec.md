# diff-extraction Specification

## Purpose
TBD - created by archiving change m0-walking-skeleton. Update Purpose after archive.
## Requirements
### Requirement: Staged diff extraction via system git
Dwarpal SHALL extract the staged diff by invoking the system `git` binary and SHALL produce a diff model containing, per file: path, change kind (added/modified/deleted/renamed), added line count, removed line count, and whether the file is binary.

#### Scenario: Modified and new files
- **WHEN** a repo has one modified file (+10/-2) and one new file (+30) staged
- **THEN** the diff model reports two entries with correct kinds and line counts

#### Scenario: Nothing staged
- **WHEN** `dwarpal check` runs with an empty staging area
- **THEN** the check passes with a "nothing to check" summary and exits 0

#### Scenario: git binary missing
- **WHEN** no `git` executable is on PATH
- **THEN** the process exits 2 with a message that system git is required

### Requirement: Edge-case handling in numstat parsing
Binary files SHALL count as 0 changed lines and 1 changed file. Renamed files SHALL be reported as renames (not delete+add). Paths containing spaces or non-ASCII characters SHALL be parsed correctly.

#### Scenario: Binary file staged
- **WHEN** an image file is staged
- **THEN** it appears in the diff model as binary with 0 line changes

#### Scenario: Rename detection
- **WHEN** a file is renamed with `git mv` and staged
- **THEN** the diff model reports one rename entry, not two entries

### Requirement: Range mode
`dwarpal check --range <a>..<b>` SHALL analyze the diff between two commits instead of the staging area, using the same diff model.

#### Scenario: Commit range
- **WHEN** `dwarpal check --range HEAD~1..HEAD` runs
- **THEN** the diff analyzed is that of the last commit, not the staging area

