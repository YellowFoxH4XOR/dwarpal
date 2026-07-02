## Context

M1 delivers the always-on deterministic gates (2, 3 regex+AST tiers, 4) plus
SARIF/Action. `spike-tree-sitter-ast` delivers `ast-engine` and `repo-index`
(function inventory + convention fingerprint), gated on proving an
incremental rebuild under the 2s budget on a ~100k-LOC repo (PRD §6.1, ROADMAP
blocker B1). M2 is the milestone where Dwarpal stops being diff-local only:
Gate 5 reads external coverage artifacts, Gate 6 and `no-duplicate-function`
read the repo-wide index the spike proves out, and `explain` plus finalized
`retry_hints` close the agent retry loop (PRD §5.1, §5.4).

This change is written against the *planned* shape of `gate-ai-patterns`
(M1) and `repo-index` (spike) as described in their proposals — those
changes have not been archived yet, so no baseline `spec.md` exists on disk
for either capability. Per the roadmap's critical path, `m2-depth-gates`
cannot implement until both land; this document and its specs describe the
target delta assuming their proposed shape holds. If either lands with a
materially different requirement set, this change's MODIFIED deltas must be
rebased before implementation starts.

Constraints carried over from the PRD/config.yaml: fail-closed for
deterministic gates (Gate 5 is deterministic in nature — it's a threshold on
parsed data — so a *malformed but present* artifact fails closed; a
*missing* artifact is warn-only per PRD §5.2 Gate 5); Gate 6 and
`no-duplicate-function` are explicitly heuristic and ship `severity: info`
by default (R4 mitigation); no network calls; p95 < 2s on a 500-line diff
still applies — coverage parsing and drift scoring must stay off the hot
path for repos where the artifact/index isn't available.

## Goals / Non-Goals

**Goals:**
- Parse the three most common coverage artifact formats (lcov, Cobertura
  XML, Go `cover.out`) into a single changed-line coverage model and gate on
  it.
- Score newly added code against a repo convention fingerprint and flag
  outliers as advisory findings, never blocking by default.
- Detect near-duplicate functions added in a diff against the existing
  `repo-index` function inventory using token-shingle similarity.
- Ship `dwarpal explain <finding-id>` so a human or agent can get the full
  rationale + doc link for any rule that fired, independent of the original
  `dwarpal check` invocation.
- Finalize the `retry_hints` schema/content against evidence from real
  Claude Code / Cursor retry loops, and apply it consistently across all M1
  + M2 gates.

**Non-Goals:**
- Running the team's test suite to produce coverage — Dwarpal only consumes
  existing artifacts (PRD §5.2 Gate 5).
- Making drift or duplicate-function block by default — advisory only in
  v1 (R4); a future config flag may allow opting into `severity: error`.
- Gate 7 (intent, LLM) and Gate 8 (plugins) — M3.
- Cross-language duplicate detection (comparing a Go function to a Python
  function) — v1 compares within the same language only.

## Decisions

**D1 — Coverage artifacts are auto-detected by well-known filename, with
explicit override.**
`gates.diff_coverage.artifact` in config points at a path; if unset, Dwarpal
probes `coverage/lcov.info`, `coverage.xml`, `cover.out` at the repo root in
that order. Format is inferred from file content (`SF:`/`DA:` header for
lcov, XML root element for Cobertura, `mode:` line for Go) not extension,
since teams rename freely. Rationale: zero-config for the common case while
staying explicit for CI. Alternative considered: require `format:` in
config always — rejected as unnecessary friction for the 90% case.

**D2 — Coverage gate maps artifact line numbers to the diff's added-line
set, not the whole file.**
Only lines the diff marks as added/modified count toward the percentage;
untouched lines in a changed file are ignored even if uncovered. This is
the PRD's explicit framing ("N% on changed lines," not file-level coverage)
and avoids penalizing agents for pre-existing debt. Implementation: build a
`map[file][]lineNo → covered bool` from the parsed artifact, intersect with
the diff's added-line ranges already computed by `diff-extraction`.

**D3 — Missing artifact is warn-only; malformed artifact is a gate error
(fail-closed).**
"No `lcov.info` found" is a normal, common state (team hasn't wired
coverage yet) — Dwarpal SHALL NOT block on absence, matching PRD §5.2 Gate 5
verbatim. But if the configured/detected artifact exists and fails to
parse (truncated, wrong format, corrupt), that is an infrastructure error
and gate-pipeline's existing fail-closed rule applies unchanged — silently
skipping a corrupt coverage file would hide real regressions.

**D4 — Convention fingerprint is computed once into `repo-index`'s cache and
scored incrementally, mirroring `no-duplicate-function`.**
Both Gate 6 and `no-duplicate-function` are `repo-index` consumers, not
producers — the spike's incremental-rebuild design is the single place that
does whole-repo tree-sitter sampling. M2 adds two new dimensions to the
index the spike didn't need: (a) per-language distributions (naming case,
import style, error-handling idiom, file-size histogram) for drift, and (b)
a shingled function inventory keyed by language for duplicate detection.
Rationale: one incremental index, two consumers — avoids each gate
re-scanning the repo and blowing the 2s budget. Alternative: gate-local
caches — rejected, duplicates the invalidation logic the spike already
solves once.

**D5 — Duplicate detection uses token-shingle Jaccard similarity over
tree-sitter function-body tokens, threshold configurable, default 0.85.**
Cheap, language-agnostic once tokenized, no embeddings/network call (stays
within the no-network constraint). Each added function's shingle set is
compared against the repo-index's function inventory *for the same
language*; best match above threshold is reported with the matched
function's location as `suggestion`. Alternative considered: AST subtree
isomorphism — more precise but expensive and harder to explain to an agent
in a `retry_hints` sentence; shingling gives a percentage a rule message
can quote directly.

**D6 — `explain` reads a static, versioned rationale table embedded via
`go:embed`, keyed by `rule_id`, not by live gate re-execution.**
`dwarpal explain <finding-id>` needs a finding to explain; since findings
aren't persisted between `check` runs, `<finding-id>` is actually
`<gate>.<rule_id>` (e.g. `diff_coverage.below_threshold`) — the explanation
is generic to the rule, not the specific past finding. Rationale/doc text
lives in `docs/rules/<rule_id>.md`, embedded at build time; `explain` looks
up by rule_id and prints rationale + `docs_url`. Alternative: persist findings
to `.dwarpal/cache/last-run.json` and let `explain` reference a specific
past finding by numeric index — rejected for v1: adds state/staleness
questions (which run? stale cache?) for marginal benefit over "explain this
rule in general," which is what the PRD's one-line description asks for.

**D7 — `retry_hints` finalized as one imperative string per finding, not
per gate.**
Testing against real agent loops (Claude Code, Cursor) showed agents act
correctly on hints attached 1:1 to the finding they fix, not on a
summarized per-gate hint — an agent fixing finding #3 shouldn't have to
parse a hint that also covers findings #1 and #2. Every finding-producing
gate (M0's diff-budget through M2's coverage/drift/duplicate) SHALL populate
`retry_hints[i]` as the i-th finding's fix instruction; `retry_hints` and
`findings` stay index-aligned arrays of equal length. This is additive to
M0/M1's existing shape, not a breaking change — M0 already populated one
hint per finding for diff-budget.

## Risks / Trade-offs

- [Coverage artifact freshness — a stale `lcov.info` from before the agent's
  last edit silently under- or over-reports] → check the artifact's mtime
  against the diff's base commit time; warn (not block) when the artifact
  predates the diff. Documented as a known limitation in the coverage
  recipes doc.
- [Drift/duplicate false positives erode trust (R4) even at `info`
  severity, if teams treat `info` noise as reason to disable Dwarpal
  entirely] → default `info`, per-rule `disable_rules`, and a suppression
  audit trail (config already supports `disable_rules`); dogfood on
  Dwarpal's own repo before calling M2 done, per the change's exit
  criterion.
- [`repo-index` convention/duplicate additions could blow the spike's proven
  budget if fingerprint sampling is naive] → sample, don't scan exhaustively
  (bounded number of files per language per rebuild); reuse the spike's
  incremental rebuild trigger (changed files only) rather than a periodic
  full rescan.
- [Hard dependency on `spike-tree-sitter-ast` and `m1-deterministic-core`
  landing first] → this change cannot start implementation until both are
  archived; tasks.md flags every task that needs `repo-index`/`ast-engine`
  explicitly so no-AST work (coverage parsing, `explain` command,
  `retry_hints` schema work) can still proceed in parallel if the spike
  slips.

## Open Questions

- Exact convention-fingerprint dimensions and outlier threshold (PRD lists
  naming/import/error-handling/file-size but not the scoring function) —
  needs a short calibration pass against 2-3 real OSS repos before Gate 6
  ships even as `info`.
- Whether `dwarpal explain` should also accept a raw `rule_id` without the
  `<gate>.` prefix for convenience, given `dwarpal rules` (M1) already lists
  `rule_id`s — leaning yes, deferred to implementation.
- Whether duplicate-function's threshold should vary by function size (tiny
  functions hit 0.85 similarity by coincidence) — likely needs a minimum
  token-count floor before comparison; to be confirmed once real repos are
  tested against `repo-index`.
