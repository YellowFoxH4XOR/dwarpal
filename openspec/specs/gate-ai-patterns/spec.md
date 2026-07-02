# gate-ai-patterns Specification

## Purpose
TBD - created by archiving change m1-deterministic-core. Update Purpose after archive.
## Requirements
### Requirement: Rule pack is data-driven and embedded
Dwarpal SHALL define built-in Gate 3 rules as data records (`{id, description, severity, languages, tier}`) compiled into the binary, not as one-off hardcoded gate implementations. Each rule SHALL declare its tier as `regex` (any language, no AST dependency) or `ast` (Go only in v1, via the stdlib `go/parser`-backed repo index; TS/JS/Python AST is future work).

#### Scenario: Rule pack lists tier per rule
- **WHEN** `dwarpal rules` is run
- **THEN** each Gate 3 rule is listed with its enabled/disabled state

### Requirement: Rules only flag lines the diff adds
Every Gate 3 rule SHALL evaluate only newly added lines in the staged diff (never pre-existing lines the commit did not touch), consistent with Dwarpal's diff-first analysis contract.

#### Scenario: Pre-existing violation not flagged
- **WHEN** a file already contains a bare `except:` clause untouched by the staged diff, and the diff only adds unrelated lines elsewhere in the same file
- **THEN** `no-broad-catch` produces no finding for the pre-existing clause

#### Scenario: Newly added violation is flagged
- **WHEN** the staged diff adds a new bare `except:` clause
- **THEN** `no-broad-catch` produces a finding on the added line

### Requirement: no-new-lint-suppressions (regex tier)
Dwarpal SHALL block newly added lint-suppression comments (`eslint-disable`, `# noqa`, `//nolint`, `@ts-ignore`, `#pragma warning disable`, and configurable equivalents) unless the commit is human-authored (per `apply_gates_to`) or carries an approved override trailer.

#### Scenario: New eslint-disable blocked
- **WHEN** an agent-authored commit adds a line containing `// eslint-disable-next-line`
- **THEN** `no-new-lint-suppressions` produces an error-severity finding on that line

#### Scenario: Override trailer suppresses the finding
- **WHEN** an agent-authored commit adds a new `# noqa` and the commit message carries the configured approved-override trailer
- **THEN** `no-new-lint-suppressions` produces no finding

### Requirement: no-hardcoded-secrets (regex tier)
Dwarpal SHALL flag newly added lines matching known secret shapes (e.g. `sk-`, `AKIA`, private-key headers) at `error` severity, and lines matching only a generic high-entropy string heuristic at `warn` severity by default.

#### Scenario: Known key shape blocked
- **WHEN** the staged diff adds a line containing a string matching the `AKIA` AWS access-key prefix pattern
- **THEN** `no-hardcoded-secrets` produces an error-severity finding

#### Scenario: Generic high-entropy string warns, does not block
- **WHEN** the staged diff adds a line containing a 40-character high-entropy string that matches no known key shape, in `enforce` mode
- **THEN** `no-hardcoded-secrets` produces a warn-severity finding and `dwarpal check` still exits 0 for that finding alone

### Requirement: no-sql-concat (AST tier, diff-local v1)
For Go, TypeScript/JavaScript, and Python, Dwarpal SHALL flag newly added string-built SQL (concatenation/interpolation into a query string) evaluated against same-file context only (no cross-package resolution in v1).

#### Scenario: String-concatenated SQL flagged
- **WHEN** the staged diff adds a Go line building a SQL query via `+` string concatenation with a variable, in a file whose other queries in the same file use parameterized placeholders
- **THEN** `no-sql-concat` produces a finding on the added line

#### Scenario: Parameterized query not flagged
- **WHEN** the staged diff adds a query built with parameterized placeholders (e.g. `db.Query("... WHERE id = ?", id)`)
- **THEN** `no-sql-concat` produces no finding

### Requirement: no-broad-catch (AST tier)
For Go, TypeScript/JavaScript, and Python, Dwarpal SHALL flag newly added bare `except:` / `catch (e) {}` blocks that swallow the error without rethrowing or logging it.

#### Scenario: Swallowed exception flagged
- **WHEN** the staged diff adds a Python `except:` block whose body is `pass`
- **THEN** `no-broad-catch` produces a finding on the added block

#### Scenario: Logged exception not flagged
- **WHEN** the staged diff adds a `catch (e) {}` block whose body calls a logging function with `e`
- **THEN** `no-broad-catch` produces no finding

### Requirement: Per-rule disable via config
`gates.ai_patterns.disable_rules` SHALL suppress the named rule(s) entirely (no findings emitted), independent of severity settings.

#### Scenario: Disabled rule produces no findings
- **WHEN** `gates.ai_patterns.disable_rules: ["no-hardcoded-secrets"]` is configured and the staged diff adds a line matching a known secret shape
- **THEN** `no-hardcoded-secrets` produces no finding and no other Gate 3 rule is affected

### Requirement: no-duplicate-function (AST tier, Go)
Gate 3 SHALL include a `no-duplicate-function` rule that detects near-duplicates of added or edited functions against the repo's existing function inventory from `repo-index`, using token-shingle similarity (identifiers and literals normalized) over Go function nodes (stdlib `go/parser`); the similarity threshold SHALL be configurable (`gates.duplicate.threshold`, default 0.85) and the rule SHALL be opt-in (`gates.duplicate.enabled`, default false) because it requires building the repo index. Findings SHALL carry `severity: warn` (heuristic honesty) and name the matched function's file and location. The rule SHALL run on Go only in v1 and SHALL be skipped, not errored, for other languages and when the repo index is unavailable.

#### Scenario: Duplicate function detected
- **WHEN** a staged diff adds a Go function whose token-shingle similarity to an existing function in `repo-index`'s inventory is at or above the configured threshold
- **THEN** the gate emits a `no-duplicate-function` finding naming the matched function's name and file, with a retry hint to reuse the existing implementation

#### Scenario: Duplicate detection skipped without repo-index
- **WHEN** the repo index is unavailable (the duplicate gate is disabled or the index was not built)
- **THEN** `no-duplicate-function` is skipped for that run without error, and the other rules in the pack still run normally

#### Scenario: Non-Go languages skipped
- **WHEN** a staged diff adds code in a language without v1 AST support (anything other than Go)
- **THEN** `no-duplicate-function` is skipped for that file without error, while regex-tier rules still evaluate it

