# SQL built by string concatenation

`ai_patterns/no-sql-concat`

## What it catches

Added SQL assembled via `+` concatenation, template-literal `${...}` (TS/JS, syntax-tree precise), or f-string interpolation (Python). Other languages use a conservative regex heuristic.

## Why this rule exists

String-built SQL is the classic injection vector, and agents reach for it far more often than parameterized queries when a codebase's conventions aren't in their context (failure mode 4).

## How to fix it

Use parameterized queries / bound placeholders. A constant query string with `?` placeholders is never flagged.


---

*`dwarpal explain no-sql-concat` shows this rationale in the terminal. False positive? `dwarpal feedback no-sql-concat --reason "..."` records it locally (never sent automatically).*
