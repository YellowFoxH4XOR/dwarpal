## Context

The v1 heuristic tier (brace/indent extractors, regex rules) shipped because no
CGO-free tree-sitter existed when the original spike was scoped. That premise
is now false: `github.com/odvcencio/gotreesitter` is a pure-Go tree-sitter
runtime (MIT, 526★, v0.20.9) with embedded grammars and full query support.
A feasibility spike (2026-07-03) confirmed on this machine:

- TS/Python/Go parse + `.scm` query, `CGO_ENABLED=0`: **works**
- First-parse latency: 7–28 ms/language (includes one-time grammar init)
- Binary with all 206 grammars embedded: **31 MB** (< 40 MB cap, §5.5)

## Goals / Non-Goals

**Goals:** real syntax trees for TS/JS/Python powering function extraction
(duplicate detection), AST-precise `no-broad-catch`/`no-sql-concat`, and the
drift import-style dimension — without giving up the static binary.

**Non-Goals:** other grammars (M4); package-context sql-concat; user
`architecture_rules` via tree-sitter queries; incremental parse caching.

## Decisions

**D1 — gotreesitter over the alternatives.** Official/smacker bindings are cgo
(breaks the §5.5 promise and the goreleaser matrix); malivvan/tree-sitter is
3 commits old. gotreesitter is pure Go, active, and query-complete. Risk: a
young-ish third-party dep — mitigated by D2's seam (swappable) and by keeping
the heuristic tier as fallback (D4).

**D2 — one thin wrapper package, `internal/astengine`.** Gates and repoindex
never import gotreesitter directly; they use `astengine.Parse(path, src)` and
`astengine.Query(tree, querySrc)`. If the dep sours, one package changes.

**D3 — languages: Go, TypeScript, JavaScript, Python only (registry-gated).**
`astengine.Supports(path)` is the single authority; everything else falls
through to existing behavior. Binary stays at ~31 MB with full embedding; a
build-tag subset is a future size optimization, not a launch requirement.

**D4 — heuristics demote to fallback, not deletion.** `repoindex.FunctionsFor`
keeps returning the heuristic extractor when astengine parsing fails (grammar
bug, pathological file). A parse failure must degrade to v1 behavior, never to
a crash or a silent skip. Go keeps stdlib `go/parser` (it is already a true
AST and faster than re-parsing via tree-sitter).

**D5 — AST-precise rules are per-language queries + small verdict functions.**
`no-broad-catch`: query catch/except clauses; flag when the handler body is
empty or contains neither a re-raise/throw nor a call (the "log or rethrow"
test). `no-sql-concat`: query binary `+` expressions and template/f-strings
whose string operand matches SQL keywords. Findings only on nodes overlapping
added lines — same diff-first discipline as everything else. The regex
heuristics continue to serve all *other* languages, and are suppressed for
files the AST tier handled (no double reporting).

**D6 — drift import-style dimension.** Fingerprint counts per language import
form (Go: grouped vs single; TS/JS: named vs default vs namespace vs require;
Python: `import` vs `from-import`). Added code whose dominant form disagrees
with a strong repo majority (>80%) gets an info finding. Computed during index
build via astengine queries (Go via go/ast, consistent with what exists).

**D7 — tolerant parsing with heuristic supplement (measured grammar gap).**
Spike matrix testing found the gotreesitter TypeScript grammar mis-parses
arrow functions with typed parameters (`(n: number) => ...`) while handling
classes, generics, interfaces, and typed declarations correctly. Typed arrows
are ubiquitous in real TS, so whole-file rejection on any error would leave
the AST tier disengaged for most TS files. Instead: a tree that parses *with
errors* is used for what it got right (captures come only from structurally
valid regions — a query match is by definition well-formed), and the heuristic
extractor supplements functions the error regions swallowed (merged by name).
Files that fail to parse at all still fall back entirely (D4).

## Risks / Trade-offs

- [Dep youth: parser bugs on real-world code] → confirmed real (typed-arrow
  gap above); mitigated by D7's tolerant+supplement strategy, D4's full
  fallback, and adversarial fixtures in the suite. Worth an upstream issue.
- [Binary grows 7.6 → ~31 MB] → within the stated cap; release size verified in
  CI going forward. Build-tag subsetting is the escape hatch if it matters.
- [Parse latency at repo scale] → re-run the #68-style benchmark with TS/Py
  corpora during implementation; index build remains opt-in-gated.

## Open Questions

- gotreesitter build-tag grammar subsetting: exact tag names undocumented —
  investigate post-launch for size optimization only.
