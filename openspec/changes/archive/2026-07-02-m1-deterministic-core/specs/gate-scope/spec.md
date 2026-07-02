## ADDED Requirements

### Requirement: Task manifest resolution precedence
Dwarpal SHALL resolve the active task manifest (declared scope paths) using this precedence, first match wins: (1) `--paths` flag passed to `dwarpal check`, (2) `.dwarpal-task.yml` present on the current branch, (3) a ticket reference parsed from the branch name or latest commit message, (4) none — no manifest found.

#### Scenario: CLI flag overrides committed manifest
- **WHEN** `dwarpal check --paths "src/auth/**"` runs in a repo that also has a `.dwarpal-task.yml` declaring `src/payments/**`
- **THEN** the active scope is `src/auth/**` only

#### Scenario: Committed manifest used when no flag given
- **WHEN** `dwarpal check` runs with no `--paths` flag and `.dwarpal-task.yml` declares `src/auth/**`
- **THEN** the active scope is `src/auth/**`

### Requirement: Out-of-scope changes blocked
When a task manifest is active, Dwarpal SHALL block any staged file change outside the declared path set, except files matching `scope.allow_always` globs.

#### Scenario: Out-of-scope file blocked
- **WHEN** the active scope is `src/auth/**` and the staged diff modifies `src/billing/invoice.go`
- **THEN** `dwarpal check` produces a `scope` finding for `src/billing/invoice.go` and exits 1 in enforce mode

#### Scenario: Always-allowed glob exempted
- **WHEN** the active scope is `src/auth/**`, `scope.allow_always` includes `**/*.lock`, and the staged diff modifies `package.lock`
- **THEN** no scope finding is produced for `package.lock`

#### Scenario: In-scope file passes
- **WHEN** the active scope is `src/auth/**` and the staged diff only modifies `src/auth/login.go`
- **THEN** no scope finding is produced

### Requirement: Warn-only when no manifest exists
When no task manifest can be resolved and `scope.require_task_manifest` is `false` (the default), Gate 4 SHALL emit no blocking finding; when `scope.require_task_manifest` is `true`, the absence of any manifest SHALL itself produce a blocking finding.

#### Scenario: No manifest, default config, no block
- **WHEN** no `--paths` flag, no `.dwarpal-task.yml`, no parseable ticket ref exists, and `scope.require_task_manifest` is unset (default `false`)
- **THEN** Gate 4 produces no finding regardless of which files changed

#### Scenario: No manifest, manifest required, blocks
- **WHEN** no manifest can be resolved and `scope.require_task_manifest: true` is configured
- **THEN** Gate 4 produces a blocking finding stating no task manifest was found
