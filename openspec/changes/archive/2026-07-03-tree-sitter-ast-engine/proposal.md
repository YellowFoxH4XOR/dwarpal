## Why

Three checklist partials (#28, #29, #37) and the AST-precise halves of #24/#25 share one root dependency: real syntax trees for TypeScript/JavaScript and Python. The v1 heuristics (brace/indent function extraction, regex sql-concat/broad-catch) deliberately shipped first, but they cannot see call expressions, import statements, or catch-clause bodies — the structures the PRD's AST tier is defined over (§5.2 Gate 3, §6 #2).

The blocker that forced the Go-only ADR is now resolved upstream: **gotreesitter** (odvcencio/gotreesitter, MIT, 526★, actively released) is a pure-Go tree-sitter runtime — no cgo, no C toolchain, full query support, selective grammar embedding. The single-static-binary promise (§5.5) survives.

## What Changes

- New `internal/astengine` package wrapping gotreesitter: language registry (Go/TS/JS/Python grammars, selectively embedded), parse cache, query runner.
- `repoindex` extractors for TS/JS and Python upgraded from brace/indent heuristics to tree-sitter function queries (heuristics retained as fallback for grammar-less builds).
- Gate 3 AST tier goes precise for TS/JS/Python: `no-broad-catch` (catch-clause body analysis: empty / no rethrow-or-log) and `no-sql-concat` (string concatenation/template-interpolation nodes containing SQL keywords) replace the regex heuristics *for those languages*; regex tier remains for all other languages.
- Gate 6 drift gains the **import-style** dimension (PRD §5.2) per language, computed from import-statement nodes in the repo fingerprint.
- Binary-size and parse-latency budgets re-verified: < 40 MB (§5.5) and p95 < 2 s (G3) with the new runtime, measured like the #68 benchmark.

## Capabilities

### New Capabilities
- `ast-engine`: pure-Go tree-sitter runtime wrapper — grammar registry, parsing, query execution (supersedes the skipped spike-change spec, adapted to the gotreesitter decision).

### Modified Capabilities
- `repo-index`: TS/JS/Python function inventory comes from tree-sitter queries (heuristic fallback documented); fingerprint gains import-style distributions.
- `gate-ai-patterns`: `no-broad-catch` and `no-sql-concat` gain an AST-precise tier for Go/TS/JS/Python; `no-duplicate-function` language list driven by the ast-engine registry.
- `gate-convention-drift`: adds the import-style outlier dimension.

## Impact

- New dependency: `github.com/odvcencio/gotreesitter` (MIT) — vetted: pure Go, active (v0.20.9), selective grammar embedding via build tags keeps binary within budget.
- `repoindex.FunctionsFor` seam means gate code is untouched by the extractor swap.
- Closes checklist #28, #29 (full), advances #24, #25, #37.

## Non-goals

- Rust/Java or other grammars (M4, demand-driven).
- The "surrounding package uses parameterized queries" context for no-sql-concat (needs cross-file query indexing — separate change if demanded).
- User-facing `architecture_rules` tree-sitter `query:` execution (the `query` config key stays accepted-but-ignored; go/ast implementation continues to serve Go).
- Incremental parse caching (#67 remains demand-driven per the benchmark).
