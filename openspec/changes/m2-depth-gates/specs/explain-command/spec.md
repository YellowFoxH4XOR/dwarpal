## ADDED Requirements

### Requirement: explain command looks up rule rationale by finding id
`dwarpal explain <finding-id>` SHALL print a human-readable rationale, the
agent failure mode it mitigates, and a `docs_url` for the named rule,
looking up `<finding-id>` in the form `<gate>.<rule_id>` against an
embedded, versioned rationale table. It SHALL work standalone, without a
prior `dwarpal check` run in the same session.

#### Scenario: Known rule
- **WHEN** `dwarpal explain diff_coverage.below_threshold` runs
- **THEN** stdout contains the rationale text and a `docs_url` pointing at
  the rule's documentation page

#### Scenario: Unknown finding id
- **WHEN** `dwarpal explain nonexistent.rule` runs
- **THEN** the process exits 2 with a message naming the unrecognized id
  and suggesting `dwarpal rules` to list valid ids

#### Scenario: Works without a prior check run
- **WHEN** `dwarpal explain` is invoked in a fresh checkout that has never
  run `dwarpal check`
- **THEN** the lookup still succeeds because the rationale table is
  embedded in the binary, not derived from a prior run's output

### Requirement: explain output is agent-consumable
`dwarpal explain <finding-id> --json` SHALL emit a JSON document with
`{rule_id, gate, rationale, failure_mode, docs_url}` and nothing else on
stdout, mirroring `check --json`'s stdout/stderr separation.

#### Scenario: JSON mode
- **WHEN** `dwarpal explain diff_coverage.below_threshold --json` runs
- **THEN** stdout parses as JSON with all five documented fields populated
  and no other stdout content
