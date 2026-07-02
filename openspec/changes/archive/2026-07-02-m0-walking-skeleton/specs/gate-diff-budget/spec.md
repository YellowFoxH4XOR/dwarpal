## ADDED Requirements

### Requirement: Line, file, and new-file budgets
The diff-budget gate SHALL block when total changed lines (added + removed), changed-file count, or new-file count exceed the configured maxima (defaults: 500 lines, 20 files, 10 new files). Each exceeded budget SHALL produce its own finding with severity `error`.

#### Scenario: Oversized diff blocked
- **WHEN** a staged diff totals 600 changed lines with `max_lines: 500`
- **THEN** the check blocks with finding `diff_budget/max-lines` stating 600 > 500

#### Scenario: Diff within budget passes
- **WHEN** a staged diff totals 100 lines across 3 files
- **THEN** the diff-budget gate emits no findings

### Requirement: Per-path-glob overrides
Budget overrides SHALL be configurable per path glob (e.g., `generated/**` allowed 10000 lines). Files matching an override glob SHALL count against that override's budget instead of the global budget. The first matching override in config order SHALL apply.

#### Scenario: Generated files exempted
- **WHEN** 2000 changed lines are all under `generated/**` with an override of `max_lines: 10000`
- **THEN** the check passes

#### Scenario: Mixed diff
- **WHEN** a diff has 400 lines in `src/**` (global budget 500) and 2000 lines in `generated/**` (override 10000)
- **THEN** the check passes because each group is within its own budget

### Requirement: Retry hints for agents
Every diff-budget finding SHALL include a `retry_hints` entry with an imperative, actionable instruction (e.g., "Split this change: 1,240 changed lines exceeds the 500-line budget").

#### Scenario: Hint present in JSON output
- **WHEN** `dwarpal check --json` blocks on max_lines
- **THEN** `retry_hints` contains an instruction mentioning both the actual and allowed line counts
