# OpenCode

```sh
dwarpal agent setup opencode       # + dwarpal init for the enforcement hooks
```

Writes a managed block into **`AGENTS.md`** (OpenCode's project instruction
convention) with the pre-flight workflow and `AGENTGATE_AGENT="OpenCode"`
identity, plus an **Agent Skill at `.agents/skills/dwarpal/SKILL.md`** — one of
the skill paths OpenCode scans natively (it also reads `.claude/skills`). The
agent loads it via its `skill` tool when working with the gate.

Enforcement layers: git hooks locally, `ci_strict` in CI. When a commit is
blocked, the JSON on stderr carries `retry_hints` the agent can act on
directly. OpenCode's plugin system could deepen this later (a pre-tool hook
like the Claude Code integration) — open an issue if you want it prioritized.
