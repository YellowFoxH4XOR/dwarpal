## ADDED Requirements

### Requirement: Import-style outlier dimension
The drift gate SHALL compare the import forms of added import statements against the repo fingerprint's per-language import distribution, and SHALL emit an info finding when the added form disagrees with a strong repo majority (dominant form ≥ 80%).

#### Scenario: require() in a named-import codebase
- **WHEN** the fingerprint shows ≥80% of TS/JS imports are ES named imports and the diff adds a `const x = require('y')` line
- **THEN** the gate emits an info finding identifying the import-style outlier and the repo's dominant form

#### Scenario: Matching import style not flagged
- **WHEN** the diff adds an ES named import in that same repo
- **THEN** the gate emits no import-style finding
