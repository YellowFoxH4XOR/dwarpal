# Function far larger than the repo norm

`convention_drift/function-size`

## What it catches

Added Go functions more than 3× the repo's average function length.

## Why this rule exists

Outlier size usually means an agent inlined what the repo decomposes. Advisory.

## How to fix it

Consider splitting to match the repo's typical granularity.


---

*`dwarpal explain function-size` shows this rationale in the terminal. False positive? `dwarpal feedback function-size --reason "..."` records it locally (never sent automatically).*
