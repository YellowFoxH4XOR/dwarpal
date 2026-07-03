# AWS access key ID committed

`ai_patterns/no-hardcoded-secrets/aws-key`

## What it catches

Added lines matching the AWS access-key shape (`AKIA` + 16 chars).

## Why this rule exists

Cloud credentials in git are the fastest route from repo to breach. GitGuardian's 2026 data shows agent-authored commits leak credentials at roughly twice the human baseline.

## How to fix it

Remove and rotate the key; use IAM roles or environment credentials.


---

*`dwarpal explain no-hardcoded-secrets/aws-key` shows this rationale in the terminal. False positive? `dwarpal feedback no-hardcoded-secrets/aws-key --reason "..."` records it locally (never sent automatically).*
