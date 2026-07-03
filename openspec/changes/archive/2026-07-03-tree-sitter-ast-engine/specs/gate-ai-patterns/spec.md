## MODIFIED Requirements

### Requirement: no-duplicate-function (AST tier, Go)
Gate 3 SHALL include a `no-duplicate-function` rule that detects near-duplicates of added or edited functions against the repo's existing function inventory from `repo-index`, using token-shingle similarity (identifiers and literals normalized) over function nodes — Go via the stdlib `go/parser`, TypeScript/JavaScript and Python via the `ast-engine` tree-sitter runtime (heuristic extraction as automatic fallback); the similarity threshold SHALL be configurable (`gates.duplicate.threshold`, default 0.85) and the rule SHALL be opt-in (`gates.duplicate.enabled`, default false) because it requires building the repo index. Findings SHALL carry `severity: warn` (heuristic honesty) and name the matched function's file and location. The rule SHALL be skipped, not errored, for languages outside the registry and when the repo index is unavailable.

#### Scenario: Duplicate function detected
- **WHEN** a staged diff adds a Go function whose token-shingle similarity to an existing function in `repo-index`'s inventory is at or above the configured threshold
- **THEN** the gate emits a `no-duplicate-function` finding naming the matched function's name and file, with a retry hint to reuse the existing implementation

#### Scenario: Duplicate detection skipped without repo-index
- **WHEN** the repo index is unavailable (the duplicate gate is disabled or the index was not built)
- **THEN** `no-duplicate-function` is skipped for that run without error, and the other rules in the pack still run normally

#### Scenario: Unsupported languages skipped
- **WHEN** a staged diff adds code in a language outside the AST registry (anything other than Go/TS/JS/Python)
- **THEN** `no-duplicate-function` is skipped for that file without error, while regex-tier rules still evaluate it

#### Scenario: TS near-duplicate detected via syntax tree
- **WHEN** a staged diff adds a TypeScript method structurally identical to an existing function (identifiers/literals renamed)
- **THEN** the gate emits a `no-duplicate-function` finding with the match's file and line from the syntax tree

### Requirement: no-broad-catch (AST tier)
For Go, TypeScript/JavaScript, and Python, Dwarpal SHALL flag newly added bare `except:` / `catch (e) {}` blocks that swallow the error without rethrowing or logging it. For TypeScript/JavaScript and Python this SHALL be evaluated over catch/except-clause syntax nodes (a handler body is compliant when it re-raises/throws or makes at least one call); the regex heuristic SHALL continue to serve other languages and SHALL NOT double-report files the AST tier handled.

#### Scenario: Swallowed exception flagged
- **WHEN** the staged diff adds a Python `except:` block whose body is `pass`
- **THEN** `no-broad-catch` produces a finding on the added block

#### Scenario: Logged exception not flagged
- **WHEN** the staged diff adds a `catch (e) {}` block whose body calls a logging function with `e`
- **THEN** `no-broad-catch` produces no finding

### Requirement: no-sql-concat (AST tier, diff-local v1)
For Go, TypeScript/JavaScript, and Python, Dwarpal SHALL flag newly added string-built SQL (concatenation/interpolation into a query string) evaluated against same-file context only (no cross-package resolution in v1). For TypeScript/JavaScript and Python this SHALL be evaluated over concatenation/template-literal/f-string syntax nodes whose string operand contains SQL keywords; the regex heuristic SHALL continue to serve other languages and SHALL NOT double-report files the AST tier handled.

#### Scenario: String-concatenated SQL flagged
- **WHEN** the staged diff adds a Go line building a SQL query via `+` string concatenation with a variable, in a file whose other queries in the same file use parameterized placeholders
- **THEN** `no-sql-concat` produces a finding on the added line

#### Scenario: Parameterized query not flagged
- **WHEN** the staged diff adds a query built with parameterized placeholders (e.g. `db.Query("... WHERE id = ?", id)`)
- **THEN** `no-sql-concat` produces no finding

#### Scenario: Template-literal SQL interpolation flagged (TS)
- **WHEN** the staged diff adds a TS template literal `` `SELECT * FROM t WHERE id = ${id}` ``
- **THEN** `no-sql-concat` produces a finding on the added line
