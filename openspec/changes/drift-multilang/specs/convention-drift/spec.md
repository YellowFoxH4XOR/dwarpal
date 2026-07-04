## MODIFIED Requirements

### Requirement: Convention drift judges naming and size per language
convention_drift SHALL score added functions' naming case and length against the
repo's convention norm FOR THAT FILE'S LANGUAGE (go, python, typescript,
javascript), using a learned per-language baseline, not a Go-centric one.

#### Scenario: Python camelCase flagged, snake_case clean
- **WHEN** a repo's Python is overwhelmingly snake_case and an added function is
  camelCase
- **THEN** it drifts; a correct snake_case addition does not

#### Scenario: Go snake_case still flagged
- **WHEN** a repo's Go is overwhelmingly camelCase and an added function is
  snake_case
- **THEN** it drifts (unchanged Go behavior)

#### Scenario: Plain lowercase word never flagged
- **WHEN** an added function is a single lowercase word (valid in both styles)
- **THEN** it does not drift regardless of the repo's dominant style
