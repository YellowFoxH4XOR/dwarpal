## Why

M2 completes the deterministic gate set. M3 adds the **two optional gates** (LLM intent, exec plugins) and the **distribution machinery** that makes Dwarpal installable in < 60 seconds (G2) and launchable. This is the PRD's M3 milestone (§10): Gate 7 (intent, BYO key), Gate 8 (plugins), Homebrew tap + goreleaser release automation, and the "Show HN" launch.

The trust promises land here: no telemetry ever, no network in default operation, diff never leaves the machine unless the user configures a remote provider.

## What Changes

- **Gate 7 — Intent Verification** (PRD §5.2, optional, off by default): task manifest + diff → LLM verdict ("accomplishes only the stated intent? surprising changes?"). Providers: Anthropic, OpenAI, OpenAI-compatible (incl. local Ollama). Hard timeout + token cap; **fail-open for LLM infra only** — a provider outage never blocks a commit (PRD §6 #6).
- **Gate 8 — Plugin Gates** (PRD §5.2): `type: exec` contract runs any command (semgrep, gitleaks, osv-scanner) against the diff; nonzero exit = findings, parsed from JSON if emitted. Turns Dwarpal into the orchestrator of a team's existing tools.
- **Distribution** (PRD §5.5): goreleaser release matrix, Homebrew tap, `go install`, curl script, Docker image, GitLab CI template, pre-commit-framework hook definition.
- **Launch collateral**: README "why harnesses beat prompts" narrative, 3 recipe posts (Claude Code, Cursor, CI-only), the "Dwarpal stopped this at the gate" brand voice.

## Capabilities

### New Capabilities
- `gate-intent`: LLM intent verification, BYO key, fail-open (PRD §5.2 Gate 7).
- `gate-plugin`: `exec` gate contract for external tools (PRD §5.2 Gate 8).
- `distribution`: goreleaser + Homebrew + Docker + CI templates (PRD §5.5).

### Modified Capabilities
- `gate-pipeline`: encode the fail-open-for-LLM-only rule (deterministic gates fail closed; intent-gate infra errors warn) — M0/M1 built fail-closed; M3 adds the single documented exception.
- `config-loading`: extend schema for `gates.intent_check` (provider/endpoint/model) and `gates.plugins`.
- `cli-core`: add `dwarpal bypass --reason` (auditable one-shot bypass; rejected under `ci_strict`) and `dwarpal doctor` (config/hook/grammar diagnostics).

## Impact

- Gate 7 is the only component that can make a network call, and only when the user opts in — the README's stated trust boundary.
- New `internal/gates/{intent,plugin}/`, `.goreleaser.yaml`, Homebrew formula, Docker/CI templates, launch docs.
- Completes all 8 gates → the PRD's positioning claim (§12) is fully backed by shipping code.

## Non-goals

- Hosted SaaS, dashboard, org management, analytics (open-core Phase 2).
- MCP server pre-flight interface (open question §11 Q4 — roadmap candidate, not v1).
- Rust/Java grammars, community rule triage (M4 — see ROADMAP.md).
