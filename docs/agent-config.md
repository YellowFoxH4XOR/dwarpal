# Let your agent author `.dwarpal.yml`

Nobody wants to hand-maintain a policy file. You already work inside a coding
agent (Claude Code, Codex, OpenCode, Pi) that has your whole repo in context —
so let *it* author and maintain `.dwarpal.yml`. Dwarpal stays deterministic and
offline; the agent is its judgment layer.

## How it works

Dwarpal never calls an LLM on your machine. Instead it ships `dwarpal analyze`,
which measures the repo deterministically and prints facts the agent turns into
a config:

```
$ dwarpal analyze
Dwarpal repo analysis (deterministic, no network) — facts for authoring .dwarpal.yml

Languages: [go]
Suggested diff_budget.max_lines: 800  (2x the 75th-percentile commit size (outlier-robust), 39 commits sampled)
  commit-size distribution: median 61, p75 397, p90 2434, max 4633 changed lines

Conventions (drift baselines):
  go:
    error idiom: wrap (60%)
    432 functions, avg 17 lines, 120 snake_case
Branch prefixes in use: [docs/ feat/]  → consider provenance.branch_prefixes
```

`analyze` makes no network call and never touches your config or source (it only
warms the same gitignored convention cache the gates use). `dwarpal analyze
--json` emits the same facts as a single JSON document for the agent to consume.

Why a *distribution* and not one number: the suggested budget is fitted to the
75th percentile of your recent commits, not the max — so one bootstrap or
generated-code commit doesn't set everyday policy. The agent sees the full
spread (median/p75/p90/max) and can override the heuristic when the tail is
skewed.

## The workflow

1. **Wire the agent once**: `dwarpal agent setup <claude-code|codex|opencode|pi>`.
   This adds a managed block to `CLAUDE.md` / `AGENTS.md` teaching the agent both
   the pre-flight commit loop *and* how to author the config from `analyze` plus
   its own reading of the codebase.
2. **Ask the agent**: "set up Dwarpal for this repo." It runs `dwarpal analyze`,
   reads your layering and generated paths, and writes a `.dwarpal.yml` that
   matches how you actually work — the analyze suggestions for budget and
   conventions, plus `architecture_rules` and `diff_budget.overrides` it can only
   infer by reading the tree.
3. **Verify**: `dwarpal rules` prints the effective ruleset; `dwarpal check` must
   still pass on a clean tree.

`dwarpal init --learn` is a shortcut: it prints the analysis first, then writes
the starter config, so you (or the agent) can tune it against real numbers.

## What stays deterministic

The gates themselves never call an LLM locally — a blocked commit is always the
result of a deterministic rule. The one optional LLM gate (`intent_check`) is
**off by default and meant for CI**, where you bring your own key; it fails open
so a provider outage never blocks a commit. Locally, your agent is the only LLM
in the loop, and it authors config rather than judging diffs.
