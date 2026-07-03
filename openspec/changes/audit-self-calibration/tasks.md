## 1. Audit engine
- [x] 1.1 internal/audit: replay recent commits through ai_patterns (scratch blob materialization per commit), capture flag records with line text
- [x] 1.2 resolve each flag vs HEAD (file gone / line rewritten = acted-on), aggregate per rule, compute recommendations
- [x] 1.3 tests: a temp repo where a flagged line is later fixed (acted-on) vs one left in place (survived)

## 2. CLI
- [x] 2.1 cmd/dwarpal/audit.go: `dwarpal audit [--window N] [--min-samples M] [--json]`; register
- [x] 2.2 audit.txtar: human + --json output, no-mutation assertion

## 3. Ship
- [ ] 3.1 full suite; docs (cli.md + CHANGELOG); PR
