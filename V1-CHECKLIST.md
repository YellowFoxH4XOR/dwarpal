# Dwarpal v1 — Master Checklist (PRD-derived)

Every requirement from dwarpal-prd.md, numbered. ✅ = shipped & verified.
☐ = pending. ◐ = partially done (what's missing noted).

## A. CLI surface (PRD §5.1)

1. ✅ `dwarpal init` — detect repo, write starter `.dwarpal.yml`, install hooks, print actions
2. ✅ `dwarpal check` against staged changes with exit codes 0/1/2
3. ✅ `dwarpal check --range <a>..<b>`
4. ✅ `dwarpal check --diff <file>` — analyze a patch file input
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
16. ✅ Configurable detection heuristics — `provenance.heuristics` regexes vs branch/message (validated at config load)
17. ✅ Block agent commits to protected branches (`main`, `release/*` globs)
18. ✅ `apply_gates_to: agent-only | all-commits` (human commits untouched by default)
19. ✅ Attach provenance as git note (refs/notes/dwarpal-provenance) on passing agent checks

## D. Gate 3 — AI-Pattern Rules (§5.2)

20. ✅ Rules-as-data pack, per-rule disable via config
21. ✅ `no-new-lint-suppressions` (eslint-disable / noqa / nolint / ts-ignore / pragma)
22. ✅ Override escape — `Dwarpal-Override:` commit trailer (--range mode) / `DWARPAL_OVERRIDE` env (staged mode)
23. ✅ `no-hardcoded-secrets` — key shapes + private-key headers + Shannon-entropy tier (URL/path false positives excluded)
24. ✅ `no-sql-concat` — AST-precise for TS/JS/Python (template-literal/f-string interpolation, `+` concat over syntax nodes); Go + other languages via the regex heuristic. Package-context resolution stays future work
25. ✅ `no-broad-catch` — AST-precise for TS/JS/Python (catch/except-clause body analysis); Go + others via regex heuristic
26. ✅ `no-duplicate-function` — token-shingle similarity vs repo inventory (Go)
27. ✅ AST language: Go (stdlib go/parser — the spike's ADR)
28. ✅ TypeScript/JavaScript — true tree-sitter AST (pure-Go gotreesitter, CGO_ENABLED=0 preserved); heuristics demoted to parse-failure fallback
29. ✅ Python — true tree-sitter AST via the same runtime; heuristic fallback on parse failure

## E. Gate 4 — Scope Enforcement (§5.2)

30. ✅ Task manifest via `dwarpal task` / `.dwarpal-task.yml`; out-of-scope files blocked
31. ✅ Ticket reference parsed from branch name (feeds intent-gate task identity)
32. ✅ Always-allowed globs (lockfiles, snapshots) + dwarpal's own files
33. ✅ Warn-only when no manifest; `require_task_manifest` to block

## F. Gate 5 — Diff Coverage (§5.2)

34. ✅ N% coverage on changed lines (default 70), lcov + cobertura + go-cover, auto-detected
35. ✅ Warn-only when artifact absent; fail-closed when malformed
36. ✅ Coverage recipes — Go, Jest, Vitest, pytest, JaCoCo, SimpleCov, coverlet (docs/recipes/coverage.md)

## G. Gate 6 — Convention Drift (§5.2)

37. ✅ Repo fingerprint — naming case + function size (Go), import-style (Go/TS/JS/Python), error-idiom (Go: wrap/bare/panic)
38. ✅ Ships `severity: info` (advisory) by default

## H. Gate 7 — Intent Verification (§5.2)

39. ✅ LLM verdict gate: off by default, BYO key (env), hard timeout, fail-open on infra error
40. ✅ OpenAI-compatible endpoint support (incl. local/Ollama)
41. ✅ Dedicated Anthropic provider (api.anthropic.com/v1/messages)
42. ✅ Task manifest id (or branch ticket ref) feeds the intent prompt

## I. Gate 8 — Plugin Gates (§5.2)

43. ✅ `type: exec` contract — any command vs the diff, nonzero exit = findings, `when:` globs
44. ✅ Structured JSON parsing for plugin tools (gitleaks/semgrep shapes), raw fallback

## J. Configuration (§5.3)

45. ✅ Versioned `.dwarpal.yml`, strict validation (unknown key → exit 2), defaults overlay
46. ✅ `mode: enforce | warn | ci_strict`
47. ✅ `architecture_rules` — user-defined forbidden-call assertions over go/ast (Go v1; `query` accepted for forward-compat)
48. ✅ `stop_on_first_block` engine knob (report-everything default)

## K. Output contract (§5.4)

49. ✅ TTY report grouped by gate, file:line, suggestions, honest blocked-vs-advisory summary
50. ✅ Stable JSON schema with imperative `retry_hints` for the agent loop
51. ✅ SARIF 2.1.0 (`--sarif`) for free GitHub PR annotation
52. ✅ `--explain-for-agent` (alias of `--json`)
53. ✅ docs_url — 25 rule pages (docs/rules/), URLs filled centrally by the engine; explain's dead docs.dwarpal.dev links replaced with the same canonical mapping

## L. Distribution & platform (§5.5)

54. ✅ Single static binary, CGO-free, darwin/linux/windows × amd64/arm64 (7.6 MB « 40 MB cap)
55. ✅ Homebrew tap (cask, auto-updated per release, fox-authored commits)
56. ✅ `go install` path
57. ✅ curl install script (with macOS quarantine self-fix)
58. ✅ GitHub Releases via goreleaser (proven twice: v0.1.0, v0.1.1)
59. ✅ Docker image — built & verified (45MB, ldflags version, mounted-repo check works, safe.directory hardening for Linux CI); registry publishing on release ☐ optional
60. ✅ GitHub Action — verified live on PR #3: install → check --sarif → SARIF upload (dogfood workflow pins the PR head SHA)
61. ✅ GitLab CI template (docs/integrations/gitlab.md)
62. ✅ pre-commit framework — .pre-commit-hooks.yaml + docs/integrations/pre-commit.md
63. ⏸ macOS codesign + notarization — pipeline wired & dormant (GoReleaser/quill); **deliberately not activated** (owner decision 2026-07-03): needs a paid Apple Developer account, and the install-script quarantine strip + `xattr` note are an acceptable stopgap. One-time activation runbook: docs/notarization.md
64. ✅ No telemetry, no network in default operation (stated in README)

## M. Engine & performance (§6)

65. ✅ Gate interface = plugin contract; Finding model; fail-closed core / fail-open LLM
66. ✅ Diff-first analysis (numstat + zero-context added-line parsing)
67. ☐ Incremental repo-index cache — **data-demoted**: eager build measured at ~150ms/140k LOC; only matters >1M LOC
68. ✅ Benchmarked: 140k-LOC index in ~150ms (13× headroom); 1.8M LOC in 2.4s; e2e check 42ms (bench kept in repo, BENCH_CORPUS-gated)
69. ✅ Parallel gate execution — concurrent gates, deterministic gate-order output; sequential under stop_on_first_block
70. ✅ Bypass resistance: pre-commit marker + pre-push verification, merge-commit-aware

## N. Repo & process

71. ✅ Dogfooding — Dwarpal gates its own repository (M1 exit criterion)
72. ✅ OpenSpec baseline: 18 capabilities / 72 requirements, all changes archived truthfully
73. ☐ Branch protection on `main` (parked by owner)
74. ✅ v0.1.0 retagged onto clean-history commit (545407f, identical tree); release + 7 assets intact, tap unaffected. NOTE: the `Akshat katiyar` name still appears on PR **merge commits** — that is the GitHub account **display name**, stamped by GitHub on merges, not a git-history issue. Owner fix: GitHub → Settings → Profile → Name → `YellowFoxH4XOR` (fixes future merges; past merge commits are immutable without a destructive full-history rewrite that breaks all tags/releases)
75. ☐ CI matrix for Windows hook behavior (PRD §11 Q3)

## O. Launch & community — G6 / M4 (§9, §10, §11)

76. ✅ Docs tree — index, full configuration reference, 25 rule pages, 4 integrations, recipes, the "why harnesses beat prompts" narrative (GitHub-rendered; a dwarpal.dev site can front it later by changing one constant)
77. ☐ Show HN + r/ClaudeCode + r/cursor + r/ExperiencedDevs launch posts
78. ☐ 3 recipe blog posts (Claude Code, Cursor, CI-only setups)
79. ☐ Register dwarpal.dev / dwarpal.io + trademark search (§11 Q1)
80. ✅ DCO chosen over CLA (ADR 0002) — CONTRIBUTING.md + DCO + a CI check enforcing sign-off
81. ✅ `dwarpal feedback <rule> --reason` — local-only log + prefilled issue URL (no-telemetry promise kept)
82. ☐ Design-partner outreach → 3 LOIs from 50+ eng companies on ci_strict (§9)
83. ☐ Next language grammar by community demand — Rust or Java (M4)
84. ☐ MCP pre-flight server evaluation for v1.x (§11 Q4)

---

**Score: 73 ✅ / 0 ◐ / 11 ☐ (84 items).**
All engineering and docs items: done. Remaining: notarization (63, needs
Apple ID), platform chores (67 demand-deferred, 74, 75), and the launch
motion (77–80, 82–84).
