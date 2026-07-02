# PRD — Dwarpal

**An open-source quality firewall between AI coding agents and your repository.**

| | |
|---|---|
| **Document status** | Draft v1.0 |
| **Date** | July 2, 2026 |
| **Owner** | Founder / Full-stack engineer |
| **Implementation language** | Go (single static binary) |
| **License (OSS core)** | Apache 2.0 |
| **Working name** | `dwarpal` — Sanskrit/Hindi for *gatekeeper*, the guardian at a temple door. Verified available: GitHub org, npm, PyPI (July 2026). TODO: confirm dwarpal.dev/.io domain + trademark search |

---

## 1. Problem Statement

AI coding agents (Claude Code, Cursor, Aider, Devin, Cline, Copilot Workspace) now author a large and growing share of all committed code — industry analyses in 2026 put AI-assisted generation at roughly 40% of new commits. The bottleneck has moved from *writing* code to *trusting* it.

Today's controls all fire **too late**:

- **AI PR reviewers** (CodeRabbit, Greptile, Qodo) review code *after* a PR exists — after the agent has already committed a 2,000-line diff nobody wants to reject.
- **SAST tools** (Semgrep, CodeQL, SonarQube) are general-purpose and agent-blind: they don't know a diff came from an agent, don't target AI-specific failure modes, and typically run in CI, not at the moment of commit.
- **Network firewalls for agents** (e.g., Pipelock) protect the *egress boundary* (credential leaks, exfiltration) — not code quality or architectural integrity.
- **Ad-hoc harnesses** — shell functions, CLAUDE.md gate protocols, hand-rolled pre-commit hooks — are what teams actually use. They are unversioned, unshared, and unenforceable across a team.

**Documented, recurring failure modes of agent-authored code** (from practitioner reports, Stanford security research, GitClear commit analysis, and high-engagement community threads):

1. Oversized, unreviewable diffs (1,500+ changed lines per PR).
2. Scope creep — files modified that have nothing to do with the task.
3. Rule-silencing — agents inserting `eslint-disable`, `# noqa`, `//nolint` to pass checks.
4. Security anti-patterns — string-concatenated SQL next to a codebase that uses parameterized queries; hardcoded "placeholder" secrets; permissive error swallowing.
5. Self-serving tests — tests that prove the agent's code works on the inputs the agent thought of, with no boundary/adversarial coverage on changed lines.
6. Convention drift — fluent-but-foreign code: naming, structure, and import patterns that don't match the repo's own distribution.
7. Unapproved dependencies — new packages pulled in to satisfy a prompt.
8. Direct commits to shared branches; missing provenance (no way to tell from `git blame` that a line was machine-generated).

**The gap:** there is no open-source, agent-agnostic, *pre-commit* enforcement layer purpose-built for agent-authored code. That is Dwarpal.

---

## 2. Product Vision

> **ESLint for the agent era.** A fast, deterministic, configurable gate pipeline that intercepts agent-generated changes at the git boundary — before commit, before push, before PR — and blocks what violates the team's quality, security, scope, and architecture rules.

One-line pitch: *"Your agents write the code. Dwarpal decides what gets in."*

Brand identity: a *dwarpal* (द्वारपाल) is the guardian figure at the entrance of a temple — the one who decides who passes through the door. Blocked commits read: **"Dwarpal stopped this at the gate."** Logo direction: a minimal, geometric temple-guardian mark.

Positioning triangle (what Dwarpal deliberately is NOT):

- Not an AI code reviewer (CodeRabbit/Greptile review PRs; we prevent bad PRs from existing).
- Not a SAST platform (we embed targeted security patterns, but Semgrep/CodeQL remain complementary and can be invoked as plugins).
- Not a network firewall (Pipelock et al. guard egress; we guard the repo).

---

## 3. Goals & Non-Goals

### Goals (v1)

| # | Goal | Measure |
|---|---|---|
| G1 | Block the top 8 agent failure modes at the git boundary | All 8 covered by built-in gates |
| G2 | Zero-friction install | `brew install dwarpal` / `go install` → `dwarpal init` in < 60 seconds |
| G3 | Deterministic-first: core gates run with no LLM, no network, no API key | Gates 1–6 pure static analysis; p95 pipeline < 2s on a 500-line diff |
| G4 | Agent-agnostic | Works with any tool that produces git changes (Claude Code, Cursor, Aider, Devin, Cline, Copilot) |
| G5 | Team-enforceable | Rules live in a versioned `.dwarpal.yml`; CI mode makes local bypass irrelevant |
| G6 | OSS traction | 2,000 GitHub stars, 50 external contributors' PRs, 200 repos with `.dwarpal.yml` on public GitHub within 6 months of launch |

### Non-Goals (v1)

- No hosted SaaS, dashboard, or org management (reserved for open-core Phase 2).
- No IDE plugins (the git boundary is the universal integration point; IDEs come later).
- No code *fixing* — Dwarpal blocks and explains; remediation is the agent's/human's job. (A `--explain-for-agent` output mode makes the block message machine-consumable so agents can self-correct.)
- No support for non-git VCS.
- No attempt to *detect* whether code is AI-authored with certainty; provenance is taken from explicit signals (branch prefix, `Co-Authored-By` trailers, env vars set by agent wrappers) with a heuristic fallback.

---

## 4. Users & Personas

**P1 — Solo builder / indie hacker ("Arjun")**
Runs Claude Code + Cursor daily; ships fast; has been burned by an agent quietly rewriting his auth middleware. Wants guardrails with zero config. Install → sane defaults → forget.

**P2 — Tech lead at a 10–50 eng startup ("Meera")**
Team adopted agents; PR review load exploded. Wants team-wide, versioned rules and a CI backstop so quality doesn't depend on each dev's local setup. Cares about scope enforcement and diff-size limits most.

**P3 — Platform/DevEx engineer at an enterprise ("Klaus")**
Rolling out agents under governance mandates (Forrester-style controls: PR gates, audit logging, provenance). Needs self-hosted everything, machine-readable audit output, SSO later. He is the future paying customer; v1 must not block his evaluation (single binary, air-gapped operation, JSON logs).

**P4 — The agent itself.**
A first-class "user": block output must be structured (JSON) so Claude Code/Cursor can read *why* the gate failed and retry correctly. This is a differentiator — gates that agents can learn from turn Dwarpal into part of the agent loop, not an obstacle to it.

---

## 5. Product Requirements

### 5.1 CLI surface (v1)

| Command | Behavior |
|---|---|
| `dwarpal init` | Detects repo language(s); writes starter `.dwarpal.yml`; installs pre-commit + pre-push hooks (via core.hooksPath, preserving existing hooks); prints what it did |
| `dwarpal check` | Runs the full gate pipeline against staged changes (or `--range <a>..<b>`, `--diff <file>`); exit 0 = pass, exit 1 = blocked, exit 2 = config/internal error |
| `dwarpal check --json` | Machine-readable result (for CI and for agents) |
| `dwarpal explain <finding-id>` | Human-readable rationale + doc link for any finding |
| `dwarpal rules` | Lists active gates/rules, source (default vs. config), and severity |
| `dwarpal bypass --reason "<text>"` | One-shot bypass; writes an auditable bypass record (git note + local log). In `ci_strict` mode, bypasses are rejected server-side |
| `dwarpal hook install / uninstall` | Manage git hooks explicitly |
| `dwarpal version / doctor` | Diagnostics: config validity, hook status, tree-sitter grammars present |

### 5.2 The Gate Pipeline

Gates run in order, cheapest first, fail-fast configurable (`stop_on_first_block: false` by default — report everything). Every gate emits findings: `{gate, rule_id, severity, file, line, message, suggestion, docs_url}`.

**Gate 1 — Diff Budget** *(deterministic)*
- Max changed lines per commit (default 500), max files (default 20), max new files (default 10). Separate budgets configurable per path glob (e.g., allow large diffs under `generated/`).
- Rationale: the single most-requested control in practitioner threads; unreviewable diffs are the root failure.

**Gate 2 — Provenance & Branch Policy** *(deterministic)*
- Require agent work on prefixed branches (`agent/*` default, configurable); block agent-flagged commits to protected branches (`main`, `release/*`).
- Detect provenance from (in order): `AGENTGATE_AGENT` env var, `Co-Authored-By:` trailers matching known agent identities, branch prefix, configurable heuristics. Attach provenance as a git note/trailer so `git blame` forensics work later.

**Gate 3 — AI-Pattern Rules** *(deterministic; tree-sitter AST + regex hybrid)*
Built-in rule pack targeting agent failure modes:
- `no-new-lint-suppressions`: block newly added `eslint-disable`, `# noqa`, `//nolint`, `@ts-ignore`, `#pragma warning disable` unless the commit is human-authored or carries an approved override trailer.
- `no-hardcoded-secrets`: entropy + pattern checks (API-key shapes, private-key headers) on added lines. (Complements, not replaces, gitleaks/trufflehog — both invocable as plugin gates.)
- `no-sql-concat`: flag string-built SQL when the surrounding package uses parameterized queries (AST query per language).
- `no-broad-catch`: newly added bare `except:` / `catch (e) {}` swallowing without rethrow/log.
- `no-duplicate-function`: near-duplicate detection of added functions vs. existing repo functions (token-shingle similarity over tree-sitter function nodes; threshold configurable).
- v1 languages for AST rules: **Go, TypeScript/JavaScript, Python** (tree-sitter grammars embedded). Regex-tier rules work on any language.

**Gate 4 — Scope Enforcement** *(deterministic)*
- Task manifest: agent (or human) declares intent — `dwarpal task "AUTH-42: password reset flow" --paths src/auth/**` or via `.dwarpal-task.yml` on the branch, or parsed from ticket reference in branch name/commit message.
- Gate blocks changes to files outside the declared path set (with configurable always-allowed globs: lockfiles, snapshots).
- If no task manifest exists: warn-only by default (configurable to block).

**Gate 5 — Diff Coverage** *(deterministic, integrates with existing coverage output)*
- Requires N% coverage **on changed lines** (default 70%), reading standard formats: `lcov.info`, `coverage.xml` (Cobertura), Go `cover.out`.
- Dwarpal does not run tests itself in v1; it consumes the artifact produced by the team's test command (documented recipes per stack). If no artifact found: warn-only.

**Gate 6 — Convention Drift** *(deterministic, heuristic)*
- Builds a lightweight fingerprint of the repo's existing conventions per language (naming case distributions, import styles, error-handling idioms, file-size norms) via tree-sitter sampling; scores added code against it; flags outliers above threshold.
- Ships as `severity: info` by default (advisory) — this gate is honest about being heuristic.

**Gate 7 — Intent Verification** *(optional, LLM, BYO key)*
- Input: task manifest + unified diff (+ optionally the spec file). Prompt: "Does this diff accomplish the stated intent, only the stated intent, and are there changes a reviewer would find surprising?" Output: structured verdict + surprise list.
- Providers: Anthropic, OpenAI, and any OpenAI-compatible endpoint incl. local (Ollama). **Off by default. No telemetry. Diff never leaves the machine unless the user configures a remote provider.**
- Hard timeout (default 30s) and token cap; failure of this gate's *infrastructure* never blocks (fail-open for the LLM gate only; all deterministic gates fail-closed).

**Gate 8 — Plugin Gates** *(deterministic contract)*
- `type: exec` gates run any command (semgrep, gitleaks, osv-scanner, custom scripts) against the diff; nonzero exit = findings (parsed from JSON if the tool emits it, else raw). This turns Dwarpal into the orchestrator of the team's existing tools at the pre-commit boundary — adoption lever, not lock-in.

### 5.3 Configuration — `.dwarpal.yml` (versioned in repo)

```yaml
version: 1
mode: enforce            # enforce | warn | ci_strict
provenance:
  branch_prefixes: ["agent/", "ai/"]
  trailers: ["Claude", "GitHub Copilot", "Cursor", "Devin", "Aider"]
  apply_gates_to: agent-only   # agent-only | all-commits

gates:
  diff_budget:
    max_lines: 500
    max_files: 20
    overrides:
      - paths: ["generated/**", "**/*.lock"]
        max_lines: 10000
  branch_policy:
    protected: ["main", "release/*"]
  ai_patterns:
    enabled: true
    disable_rules: []          # e.g. ["no-duplicate-function"]
  scope:
    require_task_manifest: false
    allow_always: ["**/*.lock", "**/__snapshots__/**"]
  diff_coverage:
    min_percent: 70
    artifact: "coverage/lcov.info"
  convention_drift:
    severity: info
  intent_check:
    enabled: false
    provider: anthropic        # anthropic | openai | openai-compatible
    endpoint: ""               # for local/self-hosted models
    model: ""
  plugins:
    - name: semgrep
      exec: "semgrep scan --json --diff"
      when: ["**/*.py", "**/*.ts"]

architecture_rules:            # user-defined AST assertions
  - id: db-through-repo-layer
    description: "No direct DB calls outside internal/repo"
    language: go
    query: "(call_expression function: (selector_expression) @call)"
    matches: "sql.Open|db.Query|db.Exec"
    forbidden_outside: ["internal/repo/**"]
    severity: error
```

### 5.4 Output contract (for humans and agents)

- Human: colored TTY report grouped by gate, with file:line, one-line fix suggestion, and `dwarpal explain <id>` pointer.
- Machine (`--json` / `--explain-for-agent`): stable schema `{result, findings[], summary, retry_hints[]}` — `retry_hints` are imperative instructions an agent can act on ("Split this change: 1,240 lines exceeds the 500-line budget. Commit auth changes separately from the refactor of pkg/util.").
- Exit codes are contract: 0 pass, 1 blocked, 2 error. CI-safe.

### 5.5 Distribution & platform requirements

- Single static Go binary; darwin/linux/windows, amd64/arm64. Target < 40 MB with embedded tree-sitter grammars (cgo-free via purego bindings or WASM grammars — spike required, see §8 risk R3).
- Install: Homebrew tap, `go install`, curl script, GitHub Releases, Docker image (for CI), GitHub Action (`uses: dwarpal/action@v1`), GitLab CI template, pre-commit-framework hook definition.
- No network calls in default operation. No telemetry in OSS build, ever (this is a stated trust promise in the README).

---

## 6. Technical Architecture (Go)

```
dwarpal/
├── cmd/dwarpal/            # CLI entrypoint (cobra)
├── internal/
│   ├── config/               # .dwarpal.yml load/validate/migrate (koanf)
│   ├── gitio/                # staged-diff extraction, ranges, notes, trailers (go-git + exec fallback)
│   ├── provenance/           # agent-detection: env, trailers, branch, heuristics
│   ├── engine/               # pipeline orchestrator: ordering, budgets, fail-fast, parallel gates
│   ├── gates/
│   │   ├── diffbudget/
│   │   ├── branchpolicy/
│   │   ├── aipatterns/       # rule pack; each rule = (matcher, languages, severity)
│   │   ├── scope/
│   │   ├── diffcoverage/     # lcov/cobertura/go-cover parsers
│   │   ├── drift/
│   │   ├── intent/           # LLM client (anthropic/openai/compatible), prompt templates
│   │   └── plugin/           # exec-gate contract
│   ├── ast/                  # tree-sitter wrapper: parse cache, query runner, language registry
│   ├── report/               # tty renderer, json encoder, sarif encoder (SARIF for CI annotation)
│   └── hooks/                # hook install/uninstall, hooksPath management, chaining
├── rules/                    # built-in rule definitions (embedded via go:embed, hot-overridable)
├── action/                   # GitHub Action wrapper
└── docs/
```

**Key engineering decisions**

1. **Diff-first analysis.** Parse only changed files (+ minimal context ring for AST queries), never the whole repo per invocation. Repo-level indices (duplicate detection, drift fingerprint) are built once into `.dwarpal/cache/` and updated incrementally — keeps the p95 < 2s budget.
2. **tree-sitter for all AST work.** One parser framework, three v1 grammars (Go, TS/JS, Python), uniform query language for both built-in and user-defined `architecture_rules`. Spike in Week 1 decides cgo vs. WASM-runtime grammars (goal: keep the binary cgo-free for painless cross-compilation).
3. **Gates are plugins internally.** Every built-in gate implements `Gate interface { ID() string; Run(ctx, *Diff, *RepoIndex) ([]Finding, error) }` — the same contract exposed to `exec` plugins. Community can contribute gates without touching the engine.
4. **go-git with shell-out fallback.** go-git for portability; fall back to invoking system `git` for operations where go-git is slow or incomplete (large staged diffs, notes).
5. **SARIF output.** Emitting SARIF gets free GitHub PR annotations in CI mode — high-leverage, low-cost.
6. **Fail-closed by default, except LLM infra.** Deterministic gate errors block; intent-gate *infrastructure* errors (timeout, provider down) warn — never let a third-party API outage stop commits.

---

## 7. Open-Core Business Model

| | OSS (Apache 2.0, forever free) | Cloud / Enterprise (Phase 2, ~month 6+) |
|---|---|---|
| All 8 gates, CLI, hooks, CI action | ✅ | ✅ |
| BYO-key intent verification | ✅ | Hosted (no key management) |
| Team analytics (rejection rates by agent/rule, quality trends, review-time saved) | — | ✅ |
| Org-wide rule registry & policy inheritance across repos | — | ✅ |
| Audit exports (SIEM), SSO/SAML, RBAC | — | ✅ |
| Bypass governance (approval workflows) | — | ✅ |

Monetization thesis: the OSS tool creates the standard and the config format; enterprises (persona P3) pay for *fleet governance over the standard* — the same motion as ESLint→enterprise lint platforms, Terraform→TFC, Semgrep OSS→Semgrep AppSec. Pricing hypothesis to validate: $15–25/dev/month for Cloud, platform deals for self-hosted enterprise.

---

## 8. Risks & Mitigations

| ID | Risk | Likelihood | Mitigation |
|---|---|---|---|
| R1 | Agent vendors (Cursor/Anthropic/GitHub) ship native guardrails | High | Agent-agnostic + team-versioned config is the moat; vendors won't build cross-vendor governance. Integrate *with* them (retry_hints consumed by Claude Code/Cursor) rather than compete |
| R2 | Hook fatigue — devs uninstall anything that slows commits | High | p95 < 2s hard budget; `warn` mode default for heuristic gates; `apply_gates_to: agent-only` default so human commits are untouched |
| R3 | tree-sitter + Go (cgo) complicates cross-compilation | Medium | Week-1 spike: cgo-free bindings vs. WASM grammars; worst case, ship cgo with prebuilt release matrix |
| R4 | False positives erode trust (esp. drift & duplicate gates) | Medium | Heuristic gates default to `info`; per-rule disable; finding IDs suppressible inline via trailer, all suppressions audited |
| R5 | "Yet another linter" perception | Medium | Positioning discipline: every rule/README line ties to a documented *agent* failure mode, not generic code quality |
| R6 | Coverage gate depends on external artifacts | Medium | Warn-only when artifact absent; ship copy-paste recipes for the top 6 stacks |
| R7 | Bypass culture (`--no-verify`) | Certain locally | That's why CI mode exists: `ci_strict` in the pipeline is the real enforcement; local hooks are DX, not security |

---

## 9. Success Metrics

**Adoption (leading):** GitHub stars (2k @ 6mo), weekly binary downloads, count of public repos containing `.dwarpal.yml` (scrapable — the honest adoption metric), GitHub Action usage count.
**Value (product):** median findings per blocked commit; % of blocks followed by a successful retry within 10 minutes (proves the agent-retry loop works); false-positive rate via `dwarpal feedback` opt-in reports; p95 pipeline latency.
**Community:** external contributors, community-contributed rules/gates merged, Discord/Discussions activity.
**Commercial (lagging, month 6+):** design-partner LOIs from ≥3 companies of 50+ engineers running `ci_strict`.

---

## 10. Milestones

**M0 — Spike (Week 1):** tree-sitter-in-Go decision (cgo vs WASM); staged-diff extraction benchmarked on 5 large OSS repos; CLI skeleton (cobra) + config loader; Gate 1 (diff budget) end-to-end with hook install. *Exit criterion: `dwarpal init && dwarpal check` blocks an oversized staged diff in < 1s.*

**M1 — Deterministic core (Weeks 2–3):** Gates 2, 3 (rules: suppressions, secrets, sql-concat, broad-catch), 4; `--json` + SARIF output; GitHub Action; docs site skeleton. *Exit: all 4 gates dogfooded on Dwarpal's own repo with Claude Code as the authoring agent.*

**M2 — Depth (Week 4):** Gates 5 (lcov + go-cover), 6 (drift, info-only), duplicate-function rule; `explain` command; retry_hints schema finalized with real Claude Code/Cursor loop testing.

**M3 — Launch (Week 5):** Gate 7 (intent, BYO key) + Gate 8 (plugins); Homebrew tap + release automation (goreleaser); README with the "why harnesses beat prompts" narrative; launch on HN ("Show HN: Dwarpal — a firewall between your coding agents and your repo"), r/ClaudeCode, r/cursor, r/ExperiencedDevs; publish 3 recipe blog posts (Claude Code, Cursor, CI-only setups).

**M4 — Listen (Weeks 6–10):** triage community rules, ship top-requested language grammar (likely Rust or Java), begin design-partner conversations for Cloud tier.

---

## 11. Open Questions

1. ~~Name clearance~~ **Resolved:** `dwarpal` verified available on GitHub org, npm, and PyPI (July 2026). Remaining: register dwarpal.dev/.io and run a formal trademark search before launch.
2. Should Gate 4 scope manifests standardize on an existing format (spec-kit task files, CLAUDE.md conventions) instead of inventing `.dwarpal-task.yml`? Leaning: support both, standardize later.
3. Windows hook behavior with GUI git clients — needs a test matrix.
4. Whether to pursue an MCP server interface in v1.x so agents can query gate rules *before* writing code (pre-flight, not just post-hoc block). Strong candidate for the roadmap — it converts Dwarpal from firewall to co-pilot-of-the-copilot.
5. Community governance: accept rule contributions under CLA or DCO?

---

## 12. Appendix — Competitive Map (July 2026)

| Product | Layer | Timing | Open source | Agent-aware | Overlap with Dwarpal |
|---|---|---|---|---|---|
| CodeRabbit / Greptile / Qodo / cubic | PR review | Post-PR | Qodo core only | Partially (reviewing AI code, not gating it) | Complementary; different moment in the lifecycle |
| Semgrep / CodeQL / SonarQube / DryRun | SAST | CI, some PR | Semgrep partly | No (agent-blind) | Invocable as Dwarpal plugins |
| Pipelock | Network egress | Runtime | Yes | Yes | None (security perimeter vs. code quality) |
| gitleaks / trufflehog | Secrets | Pre-commit/CI | Yes | No | Subsumed/complemented via plugin gate |
| Ad-hoc harnesses (blogged patterns) | Git boundary | Pre-commit | n/a | Yes | The demand signal Dwarpal productizes |

**Category claim:** Dwarpal is the first open-source, agent-agnostic **pre-commit quality firewall** for AI-authored code. Nothing in the current landscape occupies the deterministic, git-boundary, agent-aware quadrant.
