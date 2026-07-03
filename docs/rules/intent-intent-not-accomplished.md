# Intent check: diff may not accomplish the stated task

`intent/intent-not-accomplished`

## What it catches

The opt-in LLM intent gate (BYO key) judged that the diff does not accomplish the declared task intent (from `.dwarpal-task.yml` or the branch's ticket reference).

## Why this rule exists

An agent can produce a perfectly clean diff that solves the wrong problem. The intent gate is the only gate that reads *meaning* — which is also why it's advisory (`warn`), off by default, and **fail-open**: an LLM's judgment (or its provider's uptime) must never hard-block a commit.

## How to fix it

Re-read the task; either the diff is incomplete (finish it) or the verdict is wrong (advisory — proceed, and consider `dwarpal feedback intent-not-accomplished --reason "..."`).
