# Configuration reference — `.dwarpal.yml`

Lives at the repo root, versioned, so every clone shares the same policy.
**Validation is strict and fails closed**: an unknown key or invalid value
exits with code 2 naming the offender — a security gate whose misconfiguration
is silently ignored is worse than none. A missing file means compiled-in
defaults (shown below) apply.

```yaml
version: 1
mode: enforce            # enforce | warn | ci_strict
stop_on_first_block: false   # true: stop at the first blocking gate

provenance:
  branch_prefixes: ["agent/", "ai/"]
  trailers: ["Claude", "GitHub Copilot", "Cursor", "Devin", "Aider"]
  heuristics: []             # optional regexes vs branch/commit message
  apply_gates_to: all-commits # all-commits (default) | agent-only (exempt humans)

gates:
  diff_budget:
    max_lines: 500
    max_files: 20
    max_new_files: 10
    overrides:
      - paths: ["generated/**", "**/*.lock"]
        max_lines: 10000
  branch_policy:
    protected: ["main", "release/*"]
  ai_patterns:
    enabled: true
    disable_rules: []        # e.g. ["no-broad-catch"]
  scope:
    require_task_manifest: false
    allow_always: ["**/*.lock", "**/__snapshots__/**"]

rule_overrides:              # reassign a rule's severity, keyed by "gate/rule_id"
  "ai_patterns/no-broad-catch": "info"   # error | warn | info
```

`rule_overrides` reassigns any rule's severity (demoting a noisy `error` to
`warn`, or promoting an advisory rule). Author it by hand or have your agent
write it. `dwarpal rules` annotates every rule carrying an override.

## Modes

| Mode | Behavior |
|---|---|
| `enforce` | error-severity findings block (exit 1) — the default |
| `warn` | everything reported, exit is always 0 |
| `ci_strict` | enforce, plus `dwarpal bypass` **and** per-run overrides are rejected — CI is the real wall; local hooks are DX |

## Provenance & who gets gated

`apply_gates_to: all-commits` (default) gates **every commit** — the same
rules for every author. Setting `agent-only` exempts human commits: content
gates then run only when the change is detected as agent-authored, via (in
order) the `AGENTGATE_AGENT` env var, a `Co-Authored-By` trailer matching a
configured agent identity, a configured branch prefix, or a `heuristics`
regex. Branch policy always runs (it self-no-ops for humans).

## Escape hatches

All escape hatches are **rejected under `ci_strict`** — a local override carries
no authority against the CI wall.

- **Per-run rule override**: commit trailer `Dwarpal-Override: <rule-id>`
  (range/CI mode) or `DWARPAL_OVERRIDE=<rule-id>[,<rule-id>]` env (staged
  mode, where no commit message exists yet).
- **One-shot full bypass**: `dwarpal bypass --reason "..."` — arms exactly one
  commit, writes `.dwarpal/bypass.log` + a git note.
- **Policy-level disable**: `gates.ai_patterns.disable_rules` — visible in the
  versioned config's history.

## Environment variables

| Variable | Purpose |
|---|---|
| `AGENTGATE_AGENT` | Agent wrappers set this to self-identify |
| `DWARPAL_OVERRIDE` | Comma-separated rule IDs approved for this run (ignored under `ci_strict`) |
