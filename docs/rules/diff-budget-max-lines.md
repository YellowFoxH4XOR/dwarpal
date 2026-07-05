# Diff exceeds the line budget

`diff_budget/max-lines`

## What it catches

Commits whose total changed lines (added + removed) exceed `gates.diff_budget.max_lines` (default 300).

## Why this rule exists

Oversized diffs are the root agent failure mode: nobody reviews a 1,500-line PR — they skim, approve, and hope. A hard line budget forces reviewable chunks (PRD failure mode 1).

## How to fix it

Split the change into smaller, self-contained commits. If a path legitimately produces large diffs (generated code, lockfiles), add a per-glob override instead of raising the global budget.

## Configuration

```yaml
gates.diff_budget.max_lines: 300
gates.diff_budget.overrides:
  - paths: ["generated/**"]
    max_lines: 10000
```

---

*`dwarpal explain max-lines` shows this rationale in the terminal. False positive? `dwarpal feedback max-lines --reason "..."` records it locally (never sent automatically).*
