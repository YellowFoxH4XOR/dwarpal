# No task manifest declared

`scope/no-task-manifest`

## What it catches

A checked change with no `.dwarpal-task.yml` present, when `require_task_manifest: true`.

## Why this rule exists

Without declared intent there is nothing to hold the diff against. Warn-only by default; teams opt into requiring it.

## How to fix it

Declare intent: `dwarpal task AUTH-42 --paths 'src/auth/**'`.

## Configuration

```yaml
gates.scope.require_task_manifest: false
```

---

*`dwarpal explain no-task-manifest` shows this rationale in the terminal. False positive? `dwarpal feedback no-task-manifest --reason "..."` records it locally (never sent automatically).*
