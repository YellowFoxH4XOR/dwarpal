# Near-duplicate of an existing function

`ai_patterns/no-duplicate-function`

## What it catches

Added/edited functions whose normalized token-shingle similarity to an existing repo function meets `gates.duplicate.threshold` (default 0.85). Go via go/parser; TS/JS/TSX/Python via tree-sitter.

## Why this rule exists

Agents re-solve problems the codebase already solved because the existing solution wasn't in their context window (failure mode: convention drift / duplication).

## How to fix it

Reuse or extract the existing implementation named in the finding. Opt-in gate: it builds a repo function index.

## Configuration

```yaml
gates.duplicate:
  enabled: true
  threshold: 0.85
```

---

*`dwarpal explain no-duplicate-function` shows this rationale in the terminal. False positive? `dwarpal feedback no-duplicate-function --reason "..."` records it locally (never sent automatically).*
