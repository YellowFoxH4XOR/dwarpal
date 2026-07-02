## 1. Pre-flight (spike + M1 dependency gate)

- [ ] 1.1 **[BLOCKED on spike-tree-sitter-ast]** Confirm `ast-engine` and
  `repo-index` are archived with the incremental-rebuild budget proven; if
  the spike lands a materially different `repo-index` shape than assumed in
  this change's design.md, rebase the `repo-index`/`gate-ai-patterns`
  MODIFIED specs before starting section 4 or 5
- [ ] 1.2 **[BLOCKED on m1-deterministic-core]** Confirm `gate-ai-patterns`,
  `gate-pipeline`'s multi-gate registry, and `config-loading`'s extended
  schema are archived — M2's gates register into that pipeline
- [ ] 1.3 Rebase this change's `gate-ai-patterns` and `repo-index` delta
  specs against the real archived baselines (replace the assumed content
  written before either archived)

## 2. No-AST slice (can start immediately, no spike dependency)

- [ ] 2.1 Coverage artifact parsers: `internal/gates/diffcoverage/lcov.go`,
  `cobertura.go`, `gocover.go` — each parses into a common
  `map[file][]lineNo → covered bool` model
- [ ] 2.2 Format auto-detection from file content (not extension); unit
  tests per format plus an "unrecognized format" error case
- [ ] 2.3 Artifact discovery: configured path first, then default filenames
  (`coverage/lcov.info`, `coverage.xml`, `cover.out`) at repo root
- [ ] 2.4 Changed-line intersection: map parsed coverage onto the diff's
  added-line ranges (reuse `diff-extraction`'s existing per-file line data)
- [ ] 2.5 Diff-coverage gate: threshold check, missing-artifact warn-only,
  malformed-artifact fail-closed, stale-artifact info finding (mtime vs.
  diff base commit)
- [ ] 2.6 `retry_hints` for coverage findings: actual % vs. required %,
  naming the specific uncovered lines
- [ ] 2.7 txtar tests covering every `gate-diff-coverage` spec scenario
- [ ] 2.8 Coverage recipe docs for top stacks (Go, Node/Jest, Python/pytest)
  under `docs/coverage-recipes/`

## 3. explain command + retry_hints finalization (no spike dependency)

- [ ] 3.1 Rationale table: `docs/rules/<rule_id>.md` per rule (M0/M1 rules
  first, M2 rules added as their gates land), embedded via `go:embed`
- [ ] 3.2 `dwarpal explain <finding-id>` command: parse `<gate>.<rule_id>`,
  look up rationale table, print rationale + failure mode + `docs_url`
- [ ] 3.3 `--json` mode for `explain`: `{rule_id, gate, rationale,
  failure_mode, docs_url}`, stdout/stderr separation matching `check --json`
- [ ] 3.4 Exit code 2 with suggestion text on unknown finding id; exit 2 on
  missing argument
- [ ] 3.5 Register `explain` in the cobra command tree and `--help` output
- [ ] 3.6 Audit every existing gate (diff-budget, provenance, ai-patterns,
  scope from M0/M1) to confirm `retry_hints[i]` is index-aligned with
  `findings[i]`; fix any gate emitting a summarized/non-aligned hint
- [ ] 3.7 Run real Claude Code / Cursor retry loops against fixture
  failures (budget, lint-suppression, coverage) and record whether the
  agent's retry succeeds on the first attempt using only `retry_hints`;
  revise hint wording based on results
- [ ] 3.8 txtar tests covering every `explain-command` and the
  `retry_hints`-related `cli-core` spec scenarios

## 4. repo-index convention fingerprint extension **[BLOCKED on spike-tree-sitter-ast]**

- [ ] 4.1 Extend the incremental index builder with per-language
  distributions: naming case, import style, error-handling idiom, file-size
  histogram
- [ ] 4.2 Calibrate outlier scoring thresholds against 2-3 real OSS repos
  per v1 language (Go, TS/JS, Python) — resolves the design.md open question
- [ ] 4.3 Persist fingerprint alongside the existing function inventory in
  `.dwarpal/cache/`; confirm incremental rebuild still meets the 2s budget
  with the added sampling
- [ ] 4.4 Unit tests: fingerprint computation per dimension per language

## 5. Gate 6 — Convention Drift **[BLOCKED on Section 4]**

- [ ] 5.1 Drift gate: score added constructs in the diff against the
  fingerprint; emit findings for outliers above threshold
- [ ] 5.2 Default `severity: info`; honor
  `gates.convention_drift.severity` override
- [ ] 5.3 Skip-with-note behavior when no fingerprint exists yet (first run)
- [ ] 5.4 `retry_hints` for drift findings naming the repo's dominant
  convention vs. what was added
- [ ] 5.5 Register the gate in `gate-pipeline`'s ordering (cheapest-first;
  drift runs after coverage since it needs the full index)
- [ ] 5.6 txtar tests covering every `gate-convention-drift` spec scenario

## 6. no-duplicate-function rule **[BLOCKED on Section 4]**

- [ ] 6.1 Token-shingle extraction over tree-sitter function-body nodes
  (reuses `ast-engine`'s query runner)
- [ ] 6.2 Jaccard similarity comparison against `repo-index`'s function
  inventory, same-language only, configurable threshold (default 0.85)
- [ ] 6.3 Minimum token-count floor before comparing, to avoid trivial-
  function false positives (resolves design.md open question)
- [ ] 6.4 Wire into `gate-ai-patterns`'s rule pack; honor
  `disable_rules: [no-duplicate-function]`
- [ ] 6.5 Skip-with-note behavior when function inventory unavailable
- [ ] 6.6 `retry_hints` naming the matched function's file:line as the
  suggested reuse target
- [ ] 6.7 txtar tests covering the `no-duplicate-function` scenarios in the
  `gate-ai-patterns` delta spec

## 7. Config + docs

- [ ] 7.1 Extend `config-loading` schema: `gates.diff_coverage`
  (`min_percent`, `artifact`), `gates.convention_drift` (`severity`),
  `gates.ai_patterns.disable_rules` entry for `no-duplicate-function`
- [ ] 7.2 Unit tests: schema validation for the new keys (typo/out-of-domain
  cases, matching `config-loading`'s existing strict-validation pattern)
- [ ] 7.3 Update README / rule docs index to list the M2 gates and the
  `explain` command

## 8. M2 exit criterion

- [ ] 8.1 Dogfood: enable Gates 5 and 6 plus `no-duplicate-function` on
  Dwarpal's own repo; confirm no false-positive block occurs from drift or
  duplicate detection over a week of real commits
- [ ] 8.2 End-to-end acceptance: fixture repo with a real `lcov.info` below
  threshold blocks `dwarpal check`; a drift outlier and a duplicate function
  both report `info`/advisory without blocking under default config
- [ ] 8.3 Confirm `retry_hints` finalized against §3.7's agent-loop results
  and documented as the stable v1 schema
