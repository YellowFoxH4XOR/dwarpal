## ADDED Requirements

### Requirement: Built-in rules are verified against declared examples
`dwarpal rules test` SHALL verify that every built-in `ai_patterns` rule flags
each of its positive examples and flags none of its negative examples, and SHALL
exit non-zero if any rule fails or lacks examples.

#### Scenario: All rules pass
- **WHEN** `dwarpal rules test` runs and every rule's examples behave
- **THEN** it prints a per-rule table with a pass count and exits 0

#### Scenario: A broken or too-broad rule fails
- **WHEN** a rule's positive example no longer matches, or a negative example
  wrongly matches, or a rule has no examples
- **THEN** `dwarpal rules test` reports the specific failure and exits non-zero

#### Scenario: JSON output
- **WHEN** `dwarpal rules test --json` runs
- **THEN** stdout is a JSON array of per-rule results with example counts and failures
