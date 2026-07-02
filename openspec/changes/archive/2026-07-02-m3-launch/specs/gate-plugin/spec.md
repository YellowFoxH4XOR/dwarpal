## ADDED Requirements

### Requirement: exec gate contract
Each entry under `gates.plugins` SHALL declare `name` and `exec` (a shell command) and MAY declare `when` (path globs restricting which changed files trigger it). Dwarpal SHALL run the command once per `dwarpal check` invocation when at least one changed file matches `when` (or always, if `when` is absent), passing the unified diff on stdin and the list of changed files via the `DWARPAL_DIFF_FILES` environment variable (newline-separated).

#### Scenario: Plugin runs when a matching file changed
- **WHEN** a plugin is configured with `when: ["**/*.py"]` and the staged diff touches a `.py` file
- **THEN** the plugin command is executed with the diff on stdin

#### Scenario: Plugin skipped when no file matches
- **WHEN** a plugin is configured with `when: ["**/*.py"]` and the staged diff touches only `.go` files
- **THEN** the plugin command is not executed

### Requirement: Findings parsed from JSON, else nonzero exit is a single finding
If the plugin's stdout parses as the Dwarpal `Finding[]` JSON shape, those findings SHALL be used directly with `gate` rewritten to `plugin:<name>`. If stdout does not parse as that shape, a nonzero exit code SHALL produce exactly one `error`-severity finding whose message includes the exit code and a trailing excerpt of stdout/stderr. A zero exit code with unparseable stdout SHALL produce no findings.

#### Scenario: Structured JSON output consumed directly
- **WHEN** a plugin emits valid `Finding[]` JSON on stdout and exits 0
- **THEN** the report includes those findings with `gate` set to `plugin:<name>`

#### Scenario: Nonzero exit with raw output
- **WHEN** a plugin exits 1 and emits non-JSON text on stdout
- **THEN** the report includes exactly one finding for that plugin naming the exit code and containing an excerpt of the output

#### Scenario: Clean pass produces no findings
- **WHEN** a plugin exits 0 and emits output that is not valid `Finding[]` JSON
- **THEN** no findings are added for that plugin

### Requirement: Plugin gates fail closed
A plugin gate that fails to execute (binary not found, permission denied, or any error before the command runs) SHALL be treated as a gate infrastructure error under the existing "Deterministic gates fail closed" rule — it SHALL block in `enforce` mode, not be silently skipped. Gate 8 SHALL NOT receive the Gate 7 fail-open exception.

#### Scenario: Missing plugin binary blocks
- **WHEN** a configured plugin's `exec` command references a binary not present on PATH
- **THEN** `dwarpal check` in `enforce` mode exits 1 and the report names the plugin as failed to execute
