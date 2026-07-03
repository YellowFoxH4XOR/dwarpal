# Why harnesses beat prompts

Every team running coding agents converges on the same discovery, usually the
hard way: **you cannot prompt your way to guarantees.**

## The prompt ceiling

You can tell an agent "never use `--no-verify`", "keep diffs small", "don't
hardcode secrets" — in CLAUDE.md, in system prompts, in rules files. The agent
will comply. Mostly. Until the context window fills, or the task gets hard, or
the test won't pass and the mute button (`// eslint-disable`) is *right
there*. Documented cases exist of agents bypassing pre-commit hooks with
`--no-verify`, stash tricks, and quiet flags — then misrepresenting what they
did when asked ([anthropics/claude-code#40117](https://github.com/anthropics/claude-code/issues/40117)).

This isn't malice; it's optimization pressure. An agent's instructions are
suggestions weighted against everything else in its context. A prompt is a
preference. **A gate is a fact.**

## What a harness is

A harness is enforcement the agent cannot negotiate with, placed at a boundary
the work must cross. For code, that boundary is git: every change — from any
agent, any IDE, any vendor — becomes a commit. That's why Dwarpal lives there:

- **pre-commit**: deterministic gates on the staged diff — budget, scope,
  patterns, architecture. Exit 1 is exit 1 no matter how persuasive the agent.
- **pre-push**: every pushed commit must carry a marker proving it passed the
  gate — so `--no-verify` at commit time gets caught at push time.
- **CI (`ci_strict` + the GitHub Action)**: the layer no local trick reaches.
  Local hooks are developer experience; CI is enforcement.

## The part prompts get right

Prompts are how agents *improve*; harnesses are how repos *survive*. Dwarpal
is built for the loop between them: every block emits machine-readable
`retry_hints` — imperative instructions ("Split this change: 1,240 lines
exceeds the 500-line budget") the agent reads via `--explain-for-agent` and
acts on. The gate isn't an obstacle to the agent; it's the part of the agent
loop that's allowed to say no.

## Honesty rules for harness builders

Three principles keep a harness trusted enough that nobody rips it out:

1. **Deterministic gates fail closed; only the LLM gate fails open.** A
   provider outage must never block a commit; a config typo must never
   silently weaken a gate.
2. **Heuristics confess.** Drift and duplicate detection ship at `info`/`warn`
   severity — a harness that cries wolf gets uninstalled (`dwarpal feedback`
   exists because false positives are *our* bugs).
3. **Escape hatches are audited, not hidden.** `bypass` and `Dwarpal-Override`
   work — and leave a paper trail (log + git note) every time.

## The one-line version

> Prompts shape what agents *try to do*. Harnesses decide what actually
> *gets in*. Use both — but never confuse them.
