# Coverage artifact not found

`diff_coverage/artifact-missing`

## What it catches

The configured coverage artifact (`gates.diff_coverage.artifact`) doesn't exist on disk. **Informational — never blocks**: a missing artifact usually means tests haven't run yet, and a coverage gate that blocks commits when tests simply haven't run would train everyone to disable it.

## How to fix it

Run your test command with coverage output before committing (see the [coverage recipes](../recipes/coverage.md)), or run the gate in CI where the artifact is always fresh. A present-but-malformed artifact, by contrast, fails closed — corrupt data is an error, absence is a state.

---

*`dwarpal explain artifact-missing` shows this in the terminal.*
