# Dwarpal v1 — Master Checklist (PRD-derived)

Every requirement from dwarpal-prd.md, numbered. ✅ = shipped & verified.
☐ = pending. ◐ = partially done (what's missing noted).

## A. CLI surface (PRD §5.1)

1. ✅ `dwarpal init` — detect repo, write starter `.dwarpal.yml`, install hooks, print actions
2. ✅ `dwarpal check` against staged changes with exit codes 0/1/2
3. ✅ `dwarpal check --range <a>..<b>`
4. ☐ `dwarpal check --diff <file>` — analyze a patch file input
5. ✅ `dwarpal check --json` (machine contract: result/findings/summary/retry_hints)
6. ✅ `dwarpal explain <finding-id>` — rationale + fix per rule
7. ✅ `dwarpal rules` — active gates/rules and source
8. ✅ `dwarpal bypass --reason` — functional one-shot override, audited (log + git note), rejected under ci_strict
9. ✅ `dwarpal hook install / uninstall` — hooksPath management + chaining
10. ✅ `dwarpal version`
11. ✅ `dwarpal doctor` — git/config/hooks/AST diagnostics
12. ✅ `dwarpal task <id> --paths` — scope manifest declaration

## B. Gate 1 — Diff Budget (§5.2)

13. ✅ Max lines / files / new-files budgets (500/20/10 defaults)
14. ✅ Per-path-glob overrides (e.g. `generated/**`)

## C. Gate 2 — Provenance & Branch Policy (§5.2)

15. ✅ Detection: `AGENTGATE_AGENT` env → `Co-Authored-By` trailers → branch prefix, in that order
16. ☐ Configurable detection heuristics (the fourth, fallback signal)
17. ✅ Block agent commits to protected branches (`main`, `release/*` globs)
18. ✅ `apply_gates_to: agent-only | all-commits` (human commits untouched by default)
19. ☐ Attach provenance as git note/trailer for `git blame` forensics

## D. Gate 3 — AI-Pattern Rules (§5.2)

20. ✅ Rules-as-data pack, per-rule disable via config
21. ✅ `no-new-lint-suppressions` (eslint-disable / noqa / nolint / ts-ignore / pragma)
22. ☐ Approved-override-trailer escape for suppressions
23. ◐ `no-hardcoded-secrets` — key shapes + private-key headers ✅; entropy scoring ☐
24. ◐ `no-sql-concat` — diff-local heuristic ✅; AST + "package uses parameterized queries" context ☐
25. ◐ `no-broad-catch` — regex heuristic ✅; AST-precise (rethrow/log detection) ☐
26. ✅ `no-duplicate-function` — token-shingle similarity vs repo inventory (Go)
27. ✅ AST language: Go (stdlib go/parser — the spike's ADR)
28. ☐ AST language: TypeScript/JavaScript (tree-sitter via wazero-WASM)
29. ☐ AST language: Python (tree-sitter via wazero-WASM)

## E. Gate 4 — Scope Enforcement (§5.2)

30. ✅ Task manifest via `dwarpal task` / `.dwarpal-task.yml`; out-of-scope files blocked
31. ☐ Parse task reference from branch name / commit message (ticket-ID form)
32. ✅ Always-allowed globs (lockfiles, snapshots) + dwarpal's own files
33. ✅ Warn-only when no manifest; `require_task_manifest` to block

## F. Gate 5 — Diff Coverage (§5.2)

34. ✅ N% coverage on changed lines (default 70), lcov + cobertura + go-cover, auto-detected
35. ✅ Warn-only when artifact absent; fail-closed when malformed
36. ☐ Copy-paste coverage recipes for the top 6 stacks (docs)

## G. Gate 6 — Convention Drift (§5.2)

37. ◐ Repo fingerprint + outlier scoring — naming case + function size (Go) ✅; import-style and error-idiom dimensions ☐; TS/Python ☐
38. ✅ Ships `severity: info` (advisory) by default

## H. Gate 7 — Intent Verification (§5.2)

39. ✅ LLM verdict gate: off by default, BYO key (env), hard timeout, fail-open on infra error
40. ✅ OpenAI-compatible endpoint support (incl. local/Ollama)
41. ☐ Dedicated Anthropic provider
42. ☐ Feed the task manifest's intent text into the prompt (currently empty)

## I. Gate 8 — Plugin Gates (§5.2)

43. ✅ `type: exec` contract — any command vs the diff, nonzero exit = findings, `when:` globs
44. ☐ Parse structured findings from tools that emit JSON (currently raw output capture)

## J. Configuration (§5.3)

45. ✅ Versioned `.dwarpal.yml`, strict validation (unknown key → exit 2), defaults overlay
46. ✅ `mode: enforce | warn | ci_strict`
47. ☐ `architecture_rules` — user-defined AST assertions (query + forbidden_outside globs)
48. ☐ `stop_on_first_block` engine knob (engine defaults to report-everything)

## K. Output contract (§5.4)

49. ✅ TTY report grouped by gate, file:line, suggestions, honest blocked-vs-advisory summary
50. ✅ Stable JSON schema with imperative `retry_hints` for the agent loop
51. ✅ SARIF 2.1.0 (`--sarif`) for free GitHub PR annotation
52. ☐ `--explain-for-agent` named output mode (PRD's alias; `--json` covers the content)
53. ☐ Real documentation pages behind findings' `docs_url`

## L. Distribution & platform (§5.5)

54. ✅ Single static binary, CGO-free, darwin/linux/windows × amd64/arm64 (7.6 MB « 40 MB cap)
55. ✅ Homebrew tap (cask, auto-updated per release, fox-authored commits)
56. ✅ `go install` path
57. ✅ curl install script (with macOS quarantine self-fix)
58. ✅ GitHub Releases via goreleaser (proven twice: v0.1.0, v0.1.1)
59. ◐ Docker image — Dockerfile written ✅; never built/published ☐
60. ◐ GitHub Action — written + YAML-valid ✅; never exercised on a real PR ☐
61. ☐ GitLab CI template
62. ☐ pre-commit-framework hook definition
63. ☐ macOS codesign + notarization (the real Gatekeeper fix; needs Apple Developer ID)
64. ✅ No telemetry, no network in default operation (stated in README)

## M. Engine & performance (§6)

65. ✅ Gate interface = plugin contract; Finding model; fail-closed core / fail-open LLM
66. ✅ Diff-first analysis (numstat + zero-context added-line parsing)
67. ☐ Incremental repo-index cache in `.dwarpal/cache/` (v1 rebuilds eagerly, opt-in gates only)
68. ☐ Benchmark: p95 < 2s on a 500-line diff against a ~100k-LOC repo (G3's number — still unmeasured)
69. ☐ Parallel gate execution (engine is sequential; fine at current gate cost)
70. ✅ Bypass resistance: pre-commit marker + pre-push verification, merge-commit-aware

## N. Repo & process

71. ✅ Dogfooding — Dwarpal gates its own repository (M1 exit criterion)
72. ✅ OpenSpec baseline: 18 capabilities / 72 requirements, all changes archived truthfully
73. ☐ Branch protection on `main` (parked by owner)
74. ☐ Retag or retire old `v0.1.0` tag (points at pre-identity-rewrite history)
75. ☐ CI matrix for Windows hook behavior (PRD §11 Q3)

## O. Launch & community — G6 / M4 (§9, §10, §11)

76. ☐ Docs site (rationale pages, config reference, "why harnesses beat prompts")
77. ☐ Show HN + r/ClaudeCode + r/cursor + r/ExperiencedDevs launch posts
78. ☐ 3 recipe blog posts (Claude Code, Cursor, CI-only setups)
79. ☐ Register dwarpal.dev / dwarpal.io + trademark search (§11 Q1)
80. ☐ CLA vs DCO decision for contributions (§11 Q5)
81. ☐ `dwarpal feedback` opt-in false-positive reporting (§9 metrics)
82. ☐ Design-partner outreach → 3 LOIs from 50+ eng companies on ci_strict (§9)
83. ☐ Next language grammar by community demand — Rust or Java (M4)
84. ☐ MCP pre-flight server evaluation for v1.x (§11 Q4)

---

**Score: 45 ✅ / 6 ◐ / 33 ☐ (84 items).**
Functional core + distribution: done. The pending mass is in three clusters:
multi-language AST (28, 29, and the ◐ halves of 24, 25, 37), verification &
performance (59, 60, 63, 67, 68), and the entire launch motion (76–84).
