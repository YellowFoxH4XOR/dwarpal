# Self-calibration: `dwarpal audit`

## Why

The #1 reason teams disable a quality gate is **alert fatigue** — a rule that
mostly flags things nobody acts on trains developers (and agents) to ignore all
findings. Every incumbent (CodeRabbit, Greptile, Qodo, Codacy) reduces noise
with *server-side telemetry on live usage* — which a local, offline tool has no
access to. But Dwarpal has something they don't: the repo's own git history.

BitsAI-CR (ByteDance, arXiv 2501.15134) calibrates review-comment reliability
with an **"outdated rate"**: for each rule, what fraction of the lines it flagged
were later rewritten or reverted by a human. High outdated-rate = the rule
catches things people fix (signal); low = the rule flags things people leave
(noise). This is computable from git history alone — no LLM, no telemetry, no
network — which makes it a wedge move a cloud reviewer structurally cannot copy
on a local repo.

`dwarpal audit` measures this per rule and reports which rules are pulling their
weight and which are noise. It stays true to Dwarpal's core promise: fully
deterministic, offline, and — like `dwarpal analyze` — **it prints facts and
never mutates `.dwarpal.yml`**. The coding agent or a human decides what to do
with the signal.

## What changes

- New command `dwarpal audit [--window N] [--min-samples M] [--json]`: replays
  recent history through the `ai_patterns` gate, computes each rule's acted-on
  rate, and prints a table + recommendations (demote noisy rules to `warn`,
  review high-signal rules for promotion).
- New package `internal/audit`, mirroring `internal/analyze`'s deterministic,
  offline, print-only style. Reuses the existing `ai_patterns` gate and `gitio`
  diff extractor unchanged; adds only orchestration and outcome-resolution.

## Out of scope (deliberately deferred)

- **Applying** recommendations. A follow-up adds a `rule_overrides` config key,
  an engine severity-override pass, and `--apply`. Auto-*promoting* a rule to
  hard-block on a fuzzy signal is the worst failure mode, so v1 never writes
  anything — it only measures and advises.
- `convention_drift` calibration. That gate needs a full-repo fingerprint per
  historical commit (expensive); v1 calibrates `ai_patterns` only.
