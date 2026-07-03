# Changed lines under-tested

`diff_coverage/below-threshold`

## What it catches

Coverage on this change's added lines below `gates.diff_coverage.min_percent` (default 70%), read from your existing lcov/Cobertura/go-cover artifact.

## Why this rule exists

Agents write tests that prove their code works on inputs they thought of (failure mode 5). Changed-line coverage is the honest floor: new code must at least be executed by tests.

## How to fix it

Add tests exercising the added lines, regenerate the coverage artifact, and re-run. See the [coverage recipes](../recipes/coverage.md) for your stack.

## Configuration

```yaml
gates.diff_coverage:
  min_percent: 70
  artifact: coverage/lcov.info
```

---

*`dwarpal explain below-threshold` shows this rationale in the terminal. False positive? `dwarpal feedback below-threshold --reason "..."` records it locally (never sent automatically).*
