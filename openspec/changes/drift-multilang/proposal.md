# convention_drift naming & size for Python / TypeScript / JavaScript

## Why

convention_drift's naming and size dimensions were Go-only, and the naming rule
hardcoded "snake_case = drift" — correct for Go, but WRONG for Python, where
snake_case is the norm. Applied to Python it would flag correct code. Making it
multi-language requires a per-language *learned* baseline, not a switch.

## What changes

- repoindex counts function conventions PER LANGUAGE (FuncByLang), populated
  cheaply via the line-based heuristic extractor on the conventions-only hot
  path — no tree-sitter parse, so the hang-fix p95 is preserved.
- drift's naming dimension flags a function whose case bucks its language's
  strong majority (snake in a camelCase repo, or camel in a snake_case repo),
  with a dead zone for mixed repos. Size drift uses each language's own average.

## Notes

- Second of the language-parity sequence (after architecture_rules). Richer
  non-Go `analyze` output follows, reusing FuncByLang.
