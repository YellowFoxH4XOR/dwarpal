# High-entropy string looks like a secret

`ai_patterns/no-hardcoded-secrets/entropy`

## What it catches

Added tokens (≥20 chars, secret-shaped charset) whose Shannon entropy exceeds 4.0 bits/char — service-generated keys with no recognizable prefix.

## Why this rule exists

Real secrets often have no known shape; randomness is the only signal left. URLs and path-like tokens are excluded (they score high but are addresses, not credentials).

## How to fix it

Move the value to secret storage. False positive? `dwarpal feedback no-hardcoded-secrets/entropy --reason '...'` and disable per-repo if needed.

## Configuration

```yaml
gates.ai_patterns.disable_rules: ["no-hardcoded-secrets/entropy"]
```

---

*`dwarpal explain no-hardcoded-secrets/entropy` shows this rationale in the terminal. False positive? `dwarpal feedback no-hardcoded-secrets/entropy --reason "..."` records it locally (never sent automatically).*
