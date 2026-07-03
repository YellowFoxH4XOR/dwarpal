# Import form bucks the repo majority

`convention_drift/import-style`

## What it catches

Added imports whose form (require vs ES named/default/namespace; `import` vs `from-import`) disagrees with a ≥80% repo majority.

## Why this rule exists

Mixed import styles are the most visible fingerprint of paste-in agent code. Only flagged when the repo has a strong norm.

## How to fix it

Use the dominant form named in the finding.


---

*`dwarpal explain import-style` shows this rationale in the terminal. False positive? `dwarpal feedback import-style --reason "..."` records it locally (never sent automatically).*
