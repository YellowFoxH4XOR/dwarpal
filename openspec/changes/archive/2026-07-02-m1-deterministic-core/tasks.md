## 1. Engine extension (walking skeleton for many gates)

- [x] 1.1 Extend `internal/engine` registry to hold multiple ordered gates (diff-budget, branch-policy, ai-patterns, scope); freeze default registration order
- [x] 1.2 Add bounded worker-pool concurrent execution; sort aggregated findings (gate order, then file, then line) before rendering so output stays deterministic (gate-pipeline spec: "Concurrent execution preserves output determinism")
- [x] 1.3 Add `provenance.apply_gates_to` filtering in the engine: skip Gate 3/Gate 4 per-commit based on detected provenance; branch-policy (Gate 2) never filtered
- [x] 1.4 Extend config schema (`internal/config`) for `provenance`, `gates.branch_policy`, `gates.ai_patterns`, `gates.scope`; strict unknown-key/out-of-domain validation per config-loading spec
- [x] 1.5 Unit tests: partial overlay of new sections, invalid `apply_gates_to`, unknown rule ID in `disable_rules`

## 2. Gate 2 — Provenance & Branch Policy (no spike dependency)

- [x] 2.1 `internal/provenance`: detect agent identity via `AGENTGATE_AGENT` env → `Co-Authored-By:` trailer → branch prefix → heuristic fallback (off by default), first match wins
- [x] 2.2 `internal/gates/branchpolicy`: block agent-authored commits targeting configured `protected` branches
- [ ] 2.3 Attach provenance as a git note (`git notes --ref=dwarpal-provenance`) on successful `dwarpal check`, without amending the commit
- [x] 2.4 txtar tests: env var wins over trailer, branch-prefix detection, human-authored no-op, protected-branch block, non-protected pass, provenance note content

## 3. Gate 4 — Scope Enforcement (no spike dependency)

- [x] 3.1 `internal/gates/scope`: manifest resolution precedence — `--paths` flag > `.dwarpal-task.yml` > parsed ticket ref from branch/commit message > none
- [x] 3.2 Out-of-scope file blocking with `scope.allow_always` glob exemption
- [x] 3.3 Warn-only default when no manifest resolved; `scope.require_task_manifest: true` blocks on missing manifest
- [x] 3.4 `dwarpal task "<description>" --paths <glob>[,...]` CLI command: writes `.dwarpal-task.yml`; exits 2 if `--paths` missing
- [x] 3.5 txtar tests: flag overrides manifest, manifest-only resolution, out-of-scope block, always-allow exemption, in-scope pass, no-manifest warn-only, require_task_manifest block

## 4. Gate 3 — Regex tier (no spike dependency)

- [x] 4.1 `internal/gates/aipatterns`: rule-pack data model `{id, description, severity, languages, tier}`; embed via `go:embed rules/*.yml`
- [x] 4.2 Diff-line filtering: every rule only evaluates newly added lines, never pre-existing lines
- [x] 4.3 `no-new-lint-suppressions`: regex set for eslint-disable/# noqa/nolint/@ts-ignore/pragma-disable; override-trailer suppression path
- [x] 4.4 `no-hardcoded-secrets`: known key-shape regexes (error severity) + generic entropy heuristic (warn severity, tunable threshold)
- [x] 4.5 `gates.ai_patterns.disable_rules` support: suppress named rule(s) entirely
- [x] 4.6 `dwarpal rules` CLI command: list gates + Gate 3 rules with enabled/source/severity/tier columns
- [x] 4.7 txtar tests: each regex rule's block/pass scenario, disable_rules suppression, disabled-rule still listed by `dwarpal rules`

## 5. Gate 3 — AST tier (BLOCKED: spike-tree-sitter-ast)

- [ ] 5.1 BLOCKED: spike-tree-sitter-ast — adopt spike's decided tree-sitter binding (wazero/WASM or cgo) into `internal/ast`
- [ ] 5.2 BLOCKED: spike-tree-sitter-ast — `no-sql-concat` diff-local v1: tree-sitter query per language (Go/TS/JS/Python) for string-built SQL, same-file context only (PRD blocker B4 v1 scope)
- [ ] 5.3 BLOCKED: spike-tree-sitter-ast — `no-broad-catch`: tree-sitter query per language for bare `except:`/`catch (e) {}` without rethrow/log
- [ ] 5.4 BLOCKED: spike-tree-sitter-ast — txtar tests: SQL-concat flagged/parameterized-not-flagged, swallowed-exception flagged/logged-not-flagged, per language
- [ ] 5.5 BLOCKED: spike-tree-sitter-ast — `dwarpal rules`/`dwarpal doctor` report AST-tier availability (grammar present) vs. regex-tier-only fallback

## 6. SARIF output (no spike dependency)

- [x] 6.1 `internal/report`: SARIF 2.1.0 encoder as a third pure-function printer alongside `tty`/`json`, no `Finding` schema changes
- [x] 6.2 Severity → SARIF level mapping (error→error, warn→warning, info→note); rule_id → ruleId; docs_url → helpUri; file/line → physical location
- [x] 6.3 `dwarpal check --sarif <path>` flag, combinable with `--json`/default TTY without altering stdout contract
- [x] 6.4 txtar tests: SARIF written on block, severity mapping per level, file/line in physical location, `--sarif` + `--json` combined

## 7. GitHub Action (no spike dependency)

- [x] 7.1 `action/action.yml` (Docker-based) wrapping the compiled `dwarpal` binary; inputs mapped to `dwarpal check` flags (config-path, paths)
- [x] 7.2 Force `ci_strict` semantics unconditionally in Action context (overrides repo's local `mode` setting)
- [x] 7.3 Reject evidence of local bypass (missing/invalid hook-success marker) as a blocking finding in Action runs
- [x] 7.4 Upload SARIF via `github/codeql-action/upload-sarif` step
- [x] 7.5 Minimal Dockerfile sufficient to build and run the Action locally (full multi-arch release packaging deferred to M3)
- [x] 7.6 Integration test: run the Action (via `act` or a workflow fixture) against a fixture PR diff with a known violation; assert non-zero exit and SARIF upload payload

## 8. Dogfood & M1 exit criterion

- [x] 8.1 Enable Gate 2, Gate 4, and Gate 3 regex-tier in Dwarpal's own `.dwarpal.yml`; fix any findings surfaced against the existing repo
- [x] 8.2 Wire the GitHub Action into Dwarpal's own CI on PRs
- [ ] 8.3 If spike-tree-sitter-ast has closed by this point, enable Gate 3 AST tier and repeat dogfooding; otherwise record it as a known gap in README and re-run once unblocked
- [x] 8.4 End-to-end acceptance: agent-authored branch with a lint-suppression addition, an out-of-scope file, and a protected-branch push all produce the expected blocking findings in one `dwarpal check --json` run
