# Actionable calibration: `rule_overrides` + `audit --apply`

## Why

`dwarpal audit` measures which rules are noise but could only advise. This makes
the signal actionable while keeping the safety the calibration design demands:
a rule people rarely act on can be demoted so it stops blocking, but a rule is
NEVER auto-promoted to hard-block on the fuzzy acted-on signal (the worst failure
mode). Demotion only ever loosens, so it can be applied automatically; promotion
stays a human decision.

## What changes

- New config key `rule_overrides: {"<gate>/<rule_id>": "error"|"warn"|"info"}`,
  validated fail-closed, with dynamic map keys allowed under the namespace.
- The engine applies overrides to finding severity before the blocking decision
  (inside the pipeline, so a demotion is honored even in stop-on-first-block).
- `dwarpal audit --apply` writes only demotions into `rule_overrides`, preserving
  the file's comments (a yaml.Node round-trip). Promotions are reported, never
  written. `dwarpal rules` annotates each overridden rule.
