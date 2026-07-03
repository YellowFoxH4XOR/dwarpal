## MODIFIED Requirements

### Requirement: Architecture rules enforce per declared language
The architecture_rules gate SHALL enforce each rule against files of the rule's
declared `language`, supporting go, python, typescript, and javascript. A rule
whose language is not supported SHALL cause a fail-closed error, not a silent
skip.

#### Scenario: Python layering rule enforced
- **WHEN** a Python rule forbids `db.query` outside `**/repo/**` and an added
  line calls `db.query` in a non-repo `.py` file
- **THEN** the gate flags it; the same call inside `**/repo/**` is not flagged

#### Scenario: TypeScript/JavaScript layering rule enforced
- **WHEN** a rule for typescript/javascript forbids a call outside a layer and
  an added line makes that call outside it
- **THEN** the gate flags it

#### Scenario: Unsupported language fails loudly
- **WHEN** a rule declares an unsupported language
- **THEN** the gate returns an error naming the rule, rather than skipping it
