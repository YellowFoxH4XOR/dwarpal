# User-defined architecture rule violated

`architecture_rules/<your-rule-id>`

## What it catches

Added calls matching a rule's `matches` regex outside its `forbidden_outside` globs — your own layering assertions, e.g. "no direct DB calls outside internal/repo". Enforced in **Go, Python, TypeScript, and JavaScript** (each rule declares its `language`). A rule targeting an unsupported language is a loud config error, never a silent no-op.

## Why this rule exists

Architecture erodes one convenient shortcut at a time, and agents take shortcuts they can't know are forbidden. These rules make the team's layering machine-enforceable.

## How to fix it

Move the call behind the sanctioned layer (the paths in `forbidden_outside` are where it IS allowed), or amend the rule if the boundary genuinely moved.

## Configuration

```yaml
architecture_rules:
  - id: db-through-repo-layer
    description: "No direct DB calls outside internal/repo"
    language: go
    matches: "sql.Open|db.Query|db.Exec"
    forbidden_outside: ["internal/repo/**"]
    severity: error
```

---

*`dwarpal explain <your-rule-id>` shows this rationale in the terminal. False positive? `dwarpal feedback <your-rule-id> --reason "..."` records it locally (never sent automatically).*
