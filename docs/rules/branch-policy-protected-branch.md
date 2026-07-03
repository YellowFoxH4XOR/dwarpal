# Agent commit on a protected branch

`branch_policy/protected-branch`

## What it catches

Agent-authored commits targeting a protected branch (`main`, `release/*` by default).

## Why this rule exists

Direct agent commits to shared branches bypass review entirely (PRD failure mode 8). Agent work belongs on prefixed branches feeding PRs.

## How to fix it

Move the work: `git checkout -b agent/<task>` and commit there. Human commits are never blocked by this rule.

## Configuration

```yaml
gates.branch_policy.protected: ["main", "release/*"]
```

---

*`dwarpal explain protected-branch` shows this rationale in the terminal. False positive? `dwarpal feedback protected-branch --reason "..."` records it locally (never sent automatically).*
