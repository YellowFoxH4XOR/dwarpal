# Fighting codebase decay: why a diff gate isn't enough, and what to add

The problem you described — "the PR works, but long-term I accumulate dead code, unused code, duplicate code, unoptimized code, divergent patterns" — is the defining failure mode of incremental development, and it exposes a structural limit in dwarpal's current design. Dwarpal is a *diff-scoped, pre-commit* gate: it parses only the changed files and flags what the diff introduces. That is the right model for some of your list and structurally blind to the rest. The distinction that matters is **diff-local vs. whole-repo-cumulative**.

## The five decay types, classified

| Decay type | Is it a property of the diff, or the whole repo? | When does it appear? | Can a pre-commit diff gate catch it? |
|---|---|---|---|
| **Divergent patterns** (naming, imports, error idioms) | Diff-local (added code either matches the repo norm or it doesn't) | At introduction | **Yes**, this is exactly what `convention_drift` does |
| **Duplicate code** (a new function that clones an existing one) | Diff-vs-index (added function compared against the repo) | At introduction | **Yes**, `ai_patterns/no-duplicate-function` already does this |
| **Unused imports / locals** (in the code you just wrote) | Diff-local | At introduction | **Yes**, but delegate it to staticcheck/ruff via your plugin gate |
| **Dead code** (a function that becomes unreachable) | **Whole-repo reachability** | *Later*, when the last caller is removed | **No** (structurally impossible) |
| **Unused exports / "unoptimized" accumulation** | **Whole-repo** | Gradually | **No** (exports); mostly not statically decidable (optimization) |

The two rows in bold are the heart of your problem, and they are the two dwarpal cannot own with its current architecture.

## Why dead code escapes a diff gate

A function is not born dead. It becomes dead in the PR that removes its *last caller*, and that PR's diff contains only the deletion of the caller, not the now-orphaned function. Nothing in that diff textually touches the dead function itself. To know it is now unreachable, you must compute reachability over the *entire* call graph, an O(repo) analysis that cannot meet a 2-second pre-commit budget and cannot be derived from the diff alone. The same holds for unused exported symbols: a public function stops being used when its last importer changes, three files away, in a diff that never mentions it.

This is not a dwarpal bug. No pre-commit diff gate, whether dwarpal or any linter running on staged changes, can catch accumulated dead code, because deadness is a global, time-evolving property and the gate only sees a local, instantaneous slice. Recognizing this cleanly is what tells you what to build next.

## The fix: a second mode plus a ratchet

**1. Keep the pre-commit gate for what's diff-local.** Divergence and new-duplicate detection stay exactly where they are. For unused imports/locals, wire the mature deterministic detectors in as plugin gates rather than reimplementing them, since dwarpal already has the plugin-gate contract:

- Go: `staticcheck` (the `U1000` unused check) and `golang.org/x/tools/cmd/deadcode` (official, call-graph-based unreachable-function finder)
- JS/TS: `knip` (unused files, exports, dependencies — the current successor to `ts-prune`)
- Python: `vulture` (dead code), `ruff` (unused imports/vars)
- Cross-language copy-paste: `jscpd`; Go clones: `dupl`

**2. Add a whole-repo audit mode, a different command on a different cadence.** Call it `dwarpal audit`. It runs the O(repo) analyses that can't fit the commit budget: reachability-based dead code, unused-export census, and a full clone census (existing-vs-existing, not just new-vs-existing). It runs nightly or weekly in CI, not on every commit. Its output is a decay report, not a commit block.

**3. The bridge between the two modes is a ratchet, and this is the actual answer to your workflow problem.** A whole-repo audit that just prints "you have 340 dead functions" is ignored on day one, because failing the build on existing debt would block every PR. The mechanism that works is a *delta gate on a whole-repo metric*: store the audit counts (dead symbols, duplicate blocks, unused exports) as a committed baseline, and on each PR fail only if the count **went up**. "Your PR increased dead-code count by 3: `foo()`, `bar()`, `Baz.qux()`." Existing debt is grandfathered; new debt is blocked; the number can only go down over time. This is the pattern coverage ratchets and tools like `betterer` use, and it is precisely the missing piece for "the PR works but I accumulate debt": it makes each PR accountable for its *marginal* contribution to whole-repo decay, which a pure diff gate can never measure.

The ratchet also fits your existing philosophy perfectly. It is deterministic, it fails closed, it emits a concrete retry_hint ("remove these 3 now-dead symbols or justify them"), and it turns an un-actionable global metric into a per-PR, diff-attributable finding, the same move that makes your other gates work.

## "Unoptimized code": the odd one out

Of your five, "unoptimized" is the only one that is largely not statically decidable. Some specific anti-patterns are catchable as targeted rules (N+1 query loops, repeated allocation in a hot path, a linear scan where a map lookup belongs), and those belong in `ai_patterns` as named rules with example fixtures. But general "this could be faster" is a profiling-and-judgment question, not a gate; trying to make it one produces exactly the low-signal noise that erodes trust. Leave broad optimization to review and profiling; gate only the specific, named, example-tested anti-patterns.

## What this means for dwarpal concretely

You already have the harder half built (the diff gate, the plugin contract, the ratchet-friendly fail-closed/retry_hint machinery). The additions are: (a) a handful of plugin-gate presets wrapping the deterministic detectors above so users get unused/dead/duplicate detection without configuring anything, (b) a `dwarpal audit` whole-repo mode, and (c) a baseline-and-ratchet mechanism that turns the audit's global counts into per-PR delta gates. The ratchet is the novel, defensible piece: it bridges "flag the deviation this diff introduces" (what you have) and "prevent the slow accumulation of decay across many diffs" (your stated problem), and no pre-commit tool in the current landscape does it for dead/duplicate/unused code as a unified, agent-readable delta gate.
