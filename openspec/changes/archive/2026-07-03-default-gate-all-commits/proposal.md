## Why

Owner decision (2026-07-03): quality gates should apply to **every commit by
default** — rules that only bind some authors invite drift, and human-authored
mistakes (oversized diffs, hardcoded secrets) are just as costly. `agent-only`
remains as the explicit opt-out for teams that want humans exempt (the
original R2 hook-fatigue mitigation becomes a choice, not the default).

## What Changes

- **BREAKING (behavior)**: `provenance.apply_gates_to` default flips
  `agent-only` → `all-commits`. Existing configs that set the key explicitly
  are unaffected; configs relying on the old default now gate human commits.
- Starter config, README, configuration reference, and test scenarios updated.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `gate-provenance`: apply_gates_to default flipped.
- `gate-pipeline`: filtering description updated to the new default.

## Impact

- `internal/config` default; `dwarpal init` starter; docs; two txtar scenarios.

## Non-goals

- No change to detection signals, branch-policy semantics, or the escape hatches.
