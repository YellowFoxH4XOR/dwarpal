# Dwarpal documentation

**Your agents write the code. Dwarpal decides what gets in.**

Dwarpal is an open-source, agent-agnostic pre-commit quality firewall for
AI-authored code. Start with the main [README](../README.md) for install and
quickstart; this tree is the reference.

## Contents

- **[Configuration reference](configuration.md)** — every `.dwarpal.yml` key
- **[Why harnesses beat prompts](why-harnesses-beat-prompts.md)** — the design
  philosophy in one page
- **[Rule reference](rules/)** — one page per rule: what it catches, why it
  exists, how to fix findings (`dwarpal explain <rule>` shows the same
  rationale in the terminal)
- **Recipes**
  - [Coverage artifacts per stack](recipes/coverage.md) — feeding the
    diff-coverage gate from Go, Jest/Vitest, pytest, JaCoCo, SimpleCov, coverlet
- **Agents** — `dwarpal agent setup <tool>` wires the pre-flight loop
  - [Claude Code](integrations/claude-code.md) (instruction block + PreToolUse hook)
  - [Codex](integrations/codex.md) · [OpenCode](integrations/opencode.md) · [Pi](integrations/pi.md) (AGENTS.md blocks)
- **Integrations**
  - [GitHub Actions](integrations/github-actions.md)
  - [GitLab CI](integrations/gitlab.md)
  - [pre-commit framework](integrations/pre-commit.md)
  - [Docker](integrations/docker.md)
- **[macOS notarization](notarization.md)** — dormant by decision ([ADR 0001](decisions/0001-defer-macos-notarization.md)); activation runbook for later

## The gate pipeline in one paragraph

`dwarpal check` extracts the staged diff (or `--range`, or `--diff <patch>`),
detects whether the change is agent-authored (env var → `Co-Authored-By`
trailer → branch prefix → configurable heuristics), and runs the enabled gates
— diff budget, branch policy, AI-pattern rules, scope, coverage, drift,
optional LLM intent, exec plugins, and your own `architecture_rules`.
Deterministic gates fail closed; only the LLM gate fails open. Exit codes are
a contract: `0` pass, `1` blocked, `2` config/internal error. `--json` (alias
`--explain-for-agent`) emits `{result, findings, summary, retry_hints}` so the
agent that caused the block can read why and fix its own mistake.
