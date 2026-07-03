# Too many new files in one commit

`diff_budget/max-new-files`

## What it catches

Commits adding more new files than `gates.diff_budget.max_new_files` (default 10).

## Why this rule exists

Agents scaffold aggressively. A burst of new files deserves its own reviewable commit, not a ride-along.

## How to fix it

Land the scaffolding as its own commit, then the logic.

## Configuration

```yaml
gates.diff_budget.max_new_files: 10
```

---

*`dwarpal explain max-new-files` shows this rationale in the terminal. False positive? `dwarpal feedback max-new-files --reason "..."` records it locally (never sent automatically).*
