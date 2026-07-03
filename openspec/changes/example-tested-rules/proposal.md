# Example-tested rules: `dwarpal rules test`

## Why

The code-review-conformance literature converges on treating the rule set as a
first-class, versioned, **testable** spec: each rule carries positive/negative
example diffs so you can regression-test the reviewer's own judgment and defend
the false-positive budget that Google's static-analysis experience shows is
decisive (>~1-in-10 false positives → developers disable the tool). Nobody ships
this. It is the authoring-time complement to `dwarpal audit`'s runtime precision
signal: together they make the rule set a tested, calibrated artifact.

## What changes

- Each built-in `ai_patterns` rule gains `Positives` (must flag) and `Negatives`
  (must not) examples — canonical living documentation of exactly what trips it.
- New `dwarpal rules test [--json]` verifies every rule against its examples and
  exits non-zero on failure, so rule changes can be gated in CI. A rule lacking
  examples is reported as an untested gap.
- A unit test asserts full example coverage, so the "rules are tested" property
  cannot silently regress.

## Notes

- Positive examples are assembled from fragments in source so `rules.go` does not
  trip Dwarpal's own gate on commit (Dwarpal can't embed its own triggers
  verbatim).
- Scope: built-in `ai_patterns` rules. User-authored `architecture_rules`
  examples are a natural follow-up.
