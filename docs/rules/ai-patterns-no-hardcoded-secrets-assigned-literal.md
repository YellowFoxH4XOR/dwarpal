# Secret assigned as a literal

`ai_patterns/no-hardcoded-secrets/assigned-literal`

## What it catches

Added lines assigning a long literal to a secret-named variable (`api_key = "..."`, `token: '...'`).

## Why this rule exists

The variable name is the confession: values assigned to `password`/`token`/`api_key` names belong in secret storage, not source.

## How to fix it

Reference configuration or a secret manager instead of embedding the value.


---

*`dwarpal explain no-hardcoded-secrets/assigned-literal` shows this rationale in the terminal. False positive? `dwarpal feedback no-hardcoded-secrets/assigned-literal --reason "..."` records it locally (never sent automatically).*
