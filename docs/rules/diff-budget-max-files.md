# Diff touches too many files

`diff_budget/max-files`

## What it catches

Commits changing more files than `gates.diff_budget.max_files` (default 20).

## Why this rule exists

Wide diffs usually mean multiple concerns in one commit — the shape scope creep takes when an agent 'fixes' things it wasn't asked to touch.

## How to fix it

Commit each concern separately. Use `dwarpal task` to declare intended paths so the scope gate catches strays early.

## Configuration

```yaml
gates.diff_budget.max_files: 20
```

---

*`dwarpal explain max-files` shows this rationale in the terminal. False positive? `dwarpal feedback max-files --reason "..."` records it locally (never sent automatically).*
