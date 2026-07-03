# Exception swallowed silently

`ai_patterns/no-broad-catch`

## What it catches

Added `catch (e) {}` / bare `except:` handlers that neither rethrow nor make any call (TS/JS/Python: catch-clause body analysis over syntax trees; other languages: regex heuristic).

## Why this rule exists

Silent error swallowing hides failures until they become incidents. Agents add it to make red tests green (failure mode 4).

## How to fix it

Narrow the exception type and log or rethrow. Any call in the handler (e.g. `logger.error(e)`) counts as handling.


---

*`dwarpal explain no-broad-catch` shows this rationale in the terminal. False positive? `dwarpal feedback no-broad-catch --reason "..."` records it locally (never sent automatically).*
