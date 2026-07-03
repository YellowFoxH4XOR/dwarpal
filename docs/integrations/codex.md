# Codex (OpenAI Codex CLI)

```sh
dwarpal agent setup codex          # + dwarpal init for the enforcement hooks
```

Writes a managed block into **`AGENTS.md`** — the instruction file Codex reads
— teaching the pre-flight workflow: `dwarpal check --explain-for-agent`
before committing, act on `retry_hints`, export `AGENTGATE_AGENT="Codex"`,
never `--no-verify`.

Codex has no pre-tool hook mechanism, so enforcement relies on the layers
that work for any agent: Dwarpal's git pre-commit/pre-push hooks locally, and
`ci_strict` in CI. The `--explain-for-agent` JSON that a blocked commit prints
is designed to land in Codex's terminal context so it can self-correct.
