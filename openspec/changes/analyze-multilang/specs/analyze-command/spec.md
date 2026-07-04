## MODIFIED Requirements

### Requirement: Analyze fingerprints conventions per language
`dwarpal analyze` SHALL report per-language function conventions — count,
average size, and learned dominant naming style — for every supported language,
not only Go.

#### Scenario: Python fingerprint
- **WHEN** `dwarpal analyze` runs in a Python repo
- **THEN** the python conventions include a function count and a naming style of
  snake_case, in both the human output and --json
