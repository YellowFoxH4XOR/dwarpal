# Pi

```sh
dwarpal agent setup pi             # + dwarpal init for the enforcement hooks
```

Writes a managed block into **`AGENTS.md`** (which Pi reads) with the
pre-flight workflow and `AGENTGATE_AGENT="Pi"` identity, plus an **Agent Skill
at `.agents/skills/dwarpal/SKILL.md`** — the cross-agent skill path Pi reads
natively. Invoke it with `/skill:dwarpal`, or let Pi trigger it from the
description.

Enforcement layers: git hooks locally, `ci_strict` in CI; blocked commits
print `--explain-for-agent` JSON with actionable `retry_hints`.
