# External tool reported findings

`plugin/exit-nonzero`

## What it catches

A configured exec-plugin (semgrep, gitleaks, osv-scanner, any command) exited nonzero. Tools emitting JSON get per-finding file:line mapping; others surface raw output.

## Why this rule exists

Gate 8 turns Dwarpal into the orchestrator of your existing tools at the pre-commit boundary — one gate, one verdict, one retry hint for the agent.

## How to fix it

Fix what the tool reported (its output is in the finding's suggestion).

## Configuration

```yaml
gates.plugins:
  - name: gitleaks
    exec: "gitleaks protect --staged"
    when: ["**/*"]
```

---

*`dwarpal explain exit-nonzero` shows this rationale in the terminal. False positive? `dwarpal feedback exit-nonzero --reason "..."` records it locally (never sent automatically).*
