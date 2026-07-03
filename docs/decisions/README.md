# Architecture Decision Records

Significant, non-obvious decisions — the ones a contributor might otherwise
reverse by accident. Each records context, the decision, consequences, and a
"revisit when" trigger.

- [0001 — Defer macOS notarization](0001-defer-macos-notarization.md)
- [0002 — DCO over CLA for contributions](0002-dco-over-cla.md)

Other load-bearing decisions live in the roadmap and specs rather than as ADRs:
the Go-first-then-tree-sitter AST path (openspec/ROADMAP.md + the archived
`tree-sitter-ast-engine` change), the `all-commits` default flip (archived
`default-gate-all-commits`), and the gob-over-JSON index cache (its code
comments).
