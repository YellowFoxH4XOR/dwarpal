# Private key material committed

`ai_patterns/no-hardcoded-secrets/private-key`

## What it catches

Added lines containing PEM private-key headers (`-----BEGIN ... PRIVATE KEY-----`).

## Why this rule exists

Committed key material is compromised key material — history is forever, and agents paste 'placeholder' keys that are real (failure mode 4).

## How to fix it

Remove the key, rotate it (assume it leaked), and load it from a secret manager or environment variable.


---

*`dwarpal explain no-hardcoded-secrets/private-key` shows this rationale in the terminal. False positive? `dwarpal feedback no-hardcoded-secrets/private-key --reason "..."` records it locally (never sent automatically).*
