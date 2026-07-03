## ADDED Requirements

### Requirement: Severity overrides change the blocking decision
A `rule_overrides` entry SHALL reassign the severity of findings from the named
`gate/rule_id` before Dwarpal decides whether to block, so a demoted rule stops
blocking and a promoted rule starts.

#### Scenario: Demotion unblocks
- **WHEN** `rule_overrides` maps an `error` rule to `warn` and a commit trips it
- **THEN** `dwarpal check` reports the finding as advisory and does not block

#### Scenario: Invalid severity rejected
- **WHEN** a `rule_overrides` value is not error|warn|info
- **THEN** config loading fails with an error naming the offending key

### Requirement: audit --apply writes safe demotions only
`dwarpal audit --apply` SHALL write demote recommendations to `rule_overrides`,
preserving existing content and comments, and SHALL NOT auto-apply promotions.

#### Scenario: Apply demotions
- **WHEN** `dwarpal audit --apply` runs and a rule is recommended for demotion
- **THEN** a `rule_overrides` entry mapping it to `warn` is written to .dwarpal.yml,
  the file's other content and comments are preserved, and promotions (if any)
  are reported as not applied
