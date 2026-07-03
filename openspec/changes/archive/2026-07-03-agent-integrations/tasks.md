## 1. Implementation

- [x] 1.1 `internal/agentsetup`: managed-block upsert (markers, idempotent) + per-tool instruction content
- [x] 1.2 Claude Code settings.json hook merge (generic JSON, preserve unknown keys, dedupe)
- [x] 1.3 `dwarpal agent setup <tool>` command + registration
- [x] 1.4 Tests: unit (upsert idempotence, JSON merge) + txtar (all four tools, re-run idempotence, unknown tool)

## 2. Docs & ship

- [x] 2.1 docs/integrations/{claude-code,codex,opencode,pi}.md + docs index links
- [x] 2.2 README "Use inside your agent" section
- [x] 2.3 Full suite + live demo; archive change; PR
