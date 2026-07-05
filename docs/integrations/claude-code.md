# Claude Code

One command wires the full loop:

```sh
dwarpal agent setup claude-code    # + dwarpal init for the enforcement hooks
```

This does three things:

1. **`CLAUDE.md` managed block** — teaches Claude the pre-flight workflow:
   `dwarpal check --explain-for-agent` before committing, act on
   `retry_hints`, declare scope with `dwarpal task`, export
   `AGENTGATE_AGENT="Claude Code"`, never bypass.
2. **Agent Skill at `.claude/skills/dwarpal/SKILL.md`** — a native, on-demand
   skill (invoke with `/dwarpal`, or Claude auto-loads it when committing or
   editing config) with the full workflow: the pre-flight check→fix loop.
   Committed with the repo, so the whole team's Claude Code shares it.
3. **PreToolUse hook in `.claude/settings.json`** — before any `git commit`
   Bash call, the gate runs; if it blocks, the machine-readable JSON goes to
   the model as the hook's deny-reason. Claude sees *why* and fixes the
   change **before** the commit attempt, instead of parsing a failed tool
   call. Existing settings keys and hooks are preserved; re-runs are no-ops.

The three layers stack: hook feedback (best loop) → git pre-commit/pre-push
(enforcement even if instructions are ignored) → `ci_strict` + the
[GitHub Action](github-actions.md) (the wall no local trick reaches).
