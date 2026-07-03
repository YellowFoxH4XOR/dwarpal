# Function naming bucks the repo style

`convention_drift/naming-style`

## What it catches

Added Go functions using snake_case in a repo that overwhelmingly uses Go's camelCase.

## Why this rule exists

Fluent-but-foreign code (failure mode 6): correct code that reads like it was written for a different repo. Advisory (info) — honest about being a heuristic.

## How to fix it

Rename to match the repo's convention.

## Configuration

```yaml
gates.convention_drift.severity: info
```

---

*`dwarpal explain naming-style` shows this rationale in the terminal. False positive? `dwarpal feedback naming-style --reason "..."` records it locally (never sent automatically).*
