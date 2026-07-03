## ADDED Requirements

### Requirement: Deterministic per-rule self-calibration from git history
`dwarpal audit` SHALL measure, per `ai_patterns` rule, the fraction of
historically flagged lines that a human later rewrote or removed ("acted-on
rate"), using only local git history — no network, no LLM. It SHALL print the
results and SHALL NOT modify `.dwarpal.yml` or any source file (its only
permitted writes are to the gitignored cache).

#### Scenario: Audit reports acted-on rates
- **WHEN** `dwarpal audit` runs in a repo with commit history
- **THEN** it replays recent non-merge commits through the `ai_patterns` gate,
  resolves each flagged line against `HEAD` (file removed or line rewritten =
  acted-on; line still present = survived), and prints a per-rule table of
  samples, acted-on percentage, current severity, and a recommendation

#### Scenario: Noise recommendation
- **WHEN** a rule's acted-on rate is at or below the demote threshold over at
  least `--min-samples` samples
- **THEN** the rule is recommended for demotion to `warn` (a rule people don't
  act on is noise)

#### Scenario: JSON output for agents
- **WHEN** `dwarpal audit --json` runs
- **THEN** stdout is a single JSON document of the per-rule statistics

#### Scenario: No mutation
- **WHEN** `dwarpal audit` runs
- **THEN** `.dwarpal.yml` and tracked source files are left byte-for-byte
  unchanged; the command is advisory only
