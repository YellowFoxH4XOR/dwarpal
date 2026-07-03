# Error handling bucks the repo idiom

`convention_drift/error-idiom`

## What it catches

Added Go error handling (bare `return err`, `panic(...)`, or wrapped `fmt.Errorf(...%w)`) that disagrees with a ≥80% repo majority idiom.

## Why this rule exists

A repo that consistently wraps errors loses caller context every time an agent slips in a bare `return err`.

## How to fix it

Follow the dominant idiom named in the finding (usually: wrap with `fmt.Errorf("context: %w", err)`).


---

*`dwarpal explain error-idiom` shows this rationale in the terminal. False positive? `dwarpal feedback error-idiom --reason "..."` records it locally (never sent automatically).*
