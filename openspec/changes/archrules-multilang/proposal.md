# architecture_rules for Python / TypeScript / JavaScript

## Why

`architecture_rules` — Dwarpal's marquee user-authored conformance feature —
was Go-only and **silently skipped** rules for other languages. A team on a
Python or TS/JS repo could write a layering rule, see no errors, and get zero
enforcement. That silent no-op is worse than not having the feature. The
tree-sitter infrastructure to parse those languages already exists (it powers
ai_patterns' AST tier and the duplicate gate); architecture_rules just never
used it.

## What changes

- The gate enforces rules in **go, python, typescript, javascript**. For non-Go
  files it parses via the astengine and queries call expressions, using the
  captured callee text as the rendered target — the same `matches` /
  `forbidden_outside` logic as the Go path.
- A rule whose `language` is unsupported is a **fail-closed config error**
  (GateError), replacing the silent skip.

## Notes

- First of a sequence closing Go/other-language parity gaps; `convention_drift`
  (naming/size for non-Go) and richer non-Go `analyze` output follow.
