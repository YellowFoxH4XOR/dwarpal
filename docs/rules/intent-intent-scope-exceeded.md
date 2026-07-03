# Intent check: diff does more than the stated task

`intent/intent-scope-exceeded`

## What it catches

The LLM intent gate judged the diff contains changes beyond the declared intent — the semantic cousin of the scope gate's path-based check.

## Why this rule exists

Scope creep isn't always a *file* outside the manifest; sometimes it's an uninvited refactor inside an allowed file. Path globs can't see that; a reader can. Advisory and fail-open, like all intent verdicts.

## How to fix it

Split the extra work into its own commit with its own declared task.
