## MODIFIED Requirements

### Requirement: Built-in rule pack
Gate 3 SHALL ship a built-in, rules-as-data rule pack covering agent
failure modes, each rule expressed as `(matcher, languages, severity)` and
independently toggleable via `gates.ai_patterns.disable_rules`. The v1
built-in rules are:
- `no-new-lint-suppressions`: block newly added `eslint-disable`, `# noqa`,
  `//nolint`, `@ts-ignore`, `#pragma warning disable` unless the commit is
  human-authored or carries an approved override trailer.
- `no-hardcoded-secrets`: entropy + pattern checks (API-key shapes,
  private-key headers) on added lines.
- `no-sql-concat`: flag string-built SQL in added lines when the
  surrounding package uses parameterized queries elsewhere (diff-local v1).
- `no-broad-catch`: newly added bare `except:` / `catch (e) {}` swallowing
  without rethrow or log.
- `no-duplicate-function`: near-duplicate detection of added functions
  against the repo's existing function inventory (token-shingle similarity
  over tree-sitter function nodes from `repo-index`; threshold configurable,
  default 0.85).

Regex-tier rules (`no-new-lint-suppressions`, `no-hardcoded-secrets`) SHALL
work on any language. AST-tier rules (`no-sql-concat`, `no-broad-catch`,
`no-duplicate-function`) SHALL run on Go, TypeScript/JavaScript, and Python
(v1 tree-sitter grammars) and SHALL be skipped, not errored, for other
languages.

#### Scenario: Lint suppression blocked
- **WHEN** a staged diff adds a new `// nolint` comment with no override
  trailer on a non-human-authored commit
- **THEN** the gate emits a `severity: error` finding for
  `no-new-lint-suppressions`

#### Scenario: Hardcoded secret blocked
- **WHEN** a staged diff adds a line matching a known API-key shape
- **THEN** the gate emits a `severity: error` finding for
  `no-hardcoded-secrets`

#### Scenario: Broad catch blocked
- **WHEN** a staged diff adds a bare `except:` with no rethrow or log call
  in its body
- **THEN** the gate emits a `severity: error` finding for `no-broad-catch`

#### Scenario: Rule disabled via config
- **WHEN** `gates.ai_patterns.disable_rules` includes `no-broad-catch`
- **THEN** the gate never emits findings for that rule even when the
  pattern matches

#### Scenario: Duplicate function detected
- **WHEN** a staged diff adds a function whose token-shingle similarity to
  an existing function in `repo-index`'s function inventory (same language)
  is at or above the configured threshold
- **THEN** the gate emits a finding for `no-duplicate-function` naming the
  matched function's file and location as the `suggestion`

#### Scenario: Duplicate detection skipped without repo-index
- **WHEN** `repo-index`'s function inventory is unavailable (not yet built)
- **THEN** `no-duplicate-function` is skipped for that run with an
  informational note, and the other rules in the pack still run normally

#### Scenario: AST-tier rule on unsupported language
- **WHEN** a staged diff adds code in a language with no v1 tree-sitter
  grammar
- **THEN** AST-tier rules (`no-sql-concat`, `no-broad-catch`,
  `no-duplicate-function`) are skipped for that file without error, while
  regex-tier rules still evaluate it
