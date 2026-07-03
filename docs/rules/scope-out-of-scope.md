# File outside the declared task scope

`scope/out-of-scope`

## What it catches

Changed files that match neither the task manifest's declared paths nor the always-allowed globs.

## Why this rule exists

Scope creep — files modified that have nothing to do with the task — is agent failure mode 2. The manifest turns intent into an enforceable contract.

## How to fix it

Split unrelated changes into their own commit, or widen the declared paths if the file genuinely belongs: `dwarpal task <id> --paths <glob>`.

## Configuration

```yaml
gates.scope.allow_always: ["**/*.lock"]
```

---

*`dwarpal explain out-of-scope` shows this rationale in the terminal. False positive? `dwarpal feedback out-of-scope --reason "..."` records it locally (never sent automatically).*
