# CLI reference

Every command Dwarpal ships. `dwarpal <cmd> --help` prints the same detail in
your terminal. Exit codes are a contract everywhere: **0** pass, **1** blocked,
**2** config/internal error.

## `dwarpal init`

Set up Dwarpal in a repo: writes a starter `.dwarpal.yml` (never overwrites an
existing one), installs the pre-commit + pre-push hooks (chaining to any hooks
already present), and prints what it did.

## `dwarpal check`

Run the gate pipeline. Default target is the staged diff.

| Flag | Effect |
|---|---|
| `--range <a>..<b>` | check a commit range instead of the staging area |
| `--per-commit` | with `--range`: evaluate each commit's diff **separately** — budgets are per commit, so a range of compliant commits doesn't fail on their sum (this is what CI should use for PR ranges) |
| `--diff <file>` | check a unified-diff patch file instead of git |
| `--json` | machine-readable `{result, findings, summary, retry_hints}` on stdout (diagnostics stay on stderr) |
| `--explain-for-agent` | alias of `--json` — the block output an agent consumes to self-correct |
| `--sarif` | SARIF 2.1.0 for CI annotation (e.g. GitHub PR inline comments) |

## `dwarpal rules`

List the active gates and rules for this repo (from the current config) and
where each setting comes from.

### `dwarpal rules test`

Verify every built-in `ai_patterns` rule against its own positive/negative
examples: each positive must flag, each negative must stay silent, and a rule
with no examples is an untested gap. This makes the rule set a tested spec — a
regression guard on the reviewer's judgment and a false-positive-budget defense
(a negative that wrongly matches means the rule is too broad). Exits non-zero on
any failure, so it can gate rule changes in CI. `--json` for structured output.
Pairs with `dwarpal audit`: `rules test` checks a rule's *definition* against
canonical examples; `audit` checks its *precision* against your real history.

## `dwarpal analyze`

Measure the repo (conventions, a history-fitted diff budget, detected coverage
artifacts/security tools/branch prefixes/layering) and print the facts an agent
uses to author `.dwarpal.yml`. Deterministic and offline; `--json` for agents;
writes no config or source. See [agent-config](agent-config.md).

## `dwarpal audit`

Self-calibrate rules against git history. Replays recent non-merge commits
through the `ai_patterns` gate and reports, per rule, the **acted-on rate** —
the fraction of flagged lines a human later rewrote or removed. A low rate over
enough samples means the rule is noise (candidate for demotion); a high rate
means it catches things people fix. Deterministic and offline (no LLM, no
network, no telemetry), and — like `analyze` — advisory only: it prints and
never edits `.dwarpal.yml`. Flags: `--window N` (commits to replay, default
200), `--min-samples M` (default 8), `--json`, `--apply`. This is a maintenance
command, not part of the pre-commit path.

`--apply` writes only **demotions** (a noisy `error` rule → `warn`) into the
`rule_overrides` block of `.dwarpal.yml`, preserving your comments. It never
auto-*promotes* a rule to hard-block on this fuzzy signal — promotions are
surfaced for manual review only. The written overrides take effect immediately:
`dwarpal check` reads `rule_overrides` and `dwarpal rules` annotates each
overridden rule.

## `dwarpal census`

Whole-repo **decay ratchet**. Where `dwarpal check` is diff-scoped and sees only
the staged change, `census` runs configured detectors over the ENTIRE repo to
count cumulative decay — dead code, unused symbols, duplication — the kind a diff
gate structurally cannot catch (a function goes dead in a later PR whose diff
never touches it). Dwarpal owns the ratchet; the analysis is delegated to mature
external tools you install (see `--list`).

Three modes:

- **report** (no flag): print the current whole-repo counts.
- `--update-baseline`: record the current counts as the accepted baseline, a
  committed JSON file (default `.dwarpal/baseline.json`). This *grandfathers*
  existing debt.
- `--check`: fail (exit 1) only when a count went **up** versus the baseline,
  naming the net-new items. Existing debt passes; new debt blocks; the number can
  only ratchet down. Emits the same `{result, findings, retry_hints}` JSON
  contract as `dwarpal check` under `--json`.

A configured detector whose binary is **not installed** fails `--check` loudly
(exit 2) — a ratchet you could not run is never a silent pass. `--list` shows the
built-in detectors, their scope, command, and whether each is installed.

`census` is O(repo) and is **not** part of the pre-commit path — run it on its
own CI cadence (nightly, or a per-PR job). Configure it under `census:` in
`.dwarpal.yml` (see [configuration](configuration.md#census)). Diff-local
detectors can additionally be wired into `dwarpal check` via a plugin `preset:`.

## `dwarpal explain <rule-id>`

Why a rule exists and how to fix a finding it raised. Accepts the bare
`rule-id` or the `gate/rule-id` form; `--json` for structured output. Every
finding's `docs_url` points at the matching [rule page](rules/).

## `dwarpal task <id> --paths <glob>...`

Declare the current task's scope — writes `.dwarpal-task.yml`. The scope gate
then blocks files changed outside the declared paths. The task id also feeds
the intent gate (Gate 7) as the stated intent.

## `dwarpal agent setup <claude-code|codex|opencode|pi>`

Wire Dwarpal into a coding agent's own loop: an idempotent managed instruction
block in `CLAUDE.md` (Claude Code) or `AGENTS.md` (Codex/OpenCode/Pi) teaching
the pre-flight workflow, plus — for Claude Code — a PreToolUse hook merged into
`.claude/settings.json` that feeds block reasons back to the model before a
`git commit` runs. See the [agent integration guides](integrations/).

## `dwarpal bypass --reason "<text>"`

One-shot, auditable override: arms exactly **one** commit to skip the gates
(that commit still gets a push marker), and records the reason to
`.dwarpal/bypass.log` + a git note. Rejected under `mode: ci_strict`. For
skipping a *single rule* rather than the whole gate, prefer the
`Dwarpal-Override:` trailer / `DWARPAL_OVERRIDE` env (see
[configuration](configuration.md#escape-hatches-all-audited)).

## `dwarpal feedback <rule-id> --reason "<text>"`

Record a false positive **locally** (`.dwarpal/feedback.log`) and print a
prefilled GitHub-issue URL you can choose to open. Nothing is ever sent
automatically — false positives are the project's bugs, and the no-telemetry
promise is absolute.

## `dwarpal hook install | uninstall`

Manage the git hooks explicitly (what `init` does for the hook half): sets/
restores `core.hooksPath`, chains to displaced hooks, installs the marker +
pre-push verification.

## `dwarpal doctor`

Diagnose the setup without changing anything: system git presence, git
work-tree, `.dwarpal.yml` validity, hook installation, and AST language
support (probed live per language — Go via `go/parser`, TS/JS/Python via the
tree-sitter runtime). Exit 0 when the critical checks pass, else 2.

## `dwarpal version`

Version, commit, and build date (injected at release time). Handy when
several binaries are around — `version` tells you exactly which one you ran.
