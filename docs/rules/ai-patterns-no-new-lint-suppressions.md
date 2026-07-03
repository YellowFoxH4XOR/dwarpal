# New lint/type suppression added

`ai_patterns/no-new-lint-suppressions`

## What it catches

Newly added `eslint-disable`, `# noqa`, `//nolint`, `@ts-ignore`, `@ts-nocheck`, `#pragma warning disable`.

## Why this rule exists

Rule-silencing is agent failure mode 3: when a check fails, agents reach for the mute button instead of the fix.

## How to fix it

Fix the underlying warning. If a suppression is genuinely justified, a human approves it per run: commit trailer `Dwarpal-Override: no-new-lint-suppressions` (range/CI mode) or `DWARPAL_OVERRIDE=no-new-lint-suppressions` (staged mode).

## Configuration

```yaml
gates.ai_patterns.disable_rules: []   # policy-level disable (audited in config history)
```

---

*`dwarpal explain no-new-lint-suppressions` shows this rationale in the terminal. False positive? `dwarpal feedback no-new-lint-suppressions --reason "..."` records it locally (never sent automatically).*
