## ADDED Requirements

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
