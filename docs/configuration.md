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
    disable_rules: []        # e.g. ["no-hardcoded-secrets/entropy"]
  scope:
    require_task_manifest: false
    allow_always: ["**/*.lock", "**/__snapshots__/**"]
  convention_drift:
    enabled: true
    severity: info           # honest about being heuristic
  duplicate:
    enabled: false           # opt-in: builds the repo function index
    threshold: 0.85          # Jaccard similarity cutoff
  diff_coverage:             # active only when artifact is set
    min_percent: 70
    artifact: "coverage/lcov.info"   # lcov / cobertura XML / go cover.out
  intent_check:              # LLM gate — off by default, BYO key, fail-open
    enabled: false
    provider: openai-compatible     # anthropic | openai-compatible
    endpoint: ""             # for local/self-hosted (e.g. Ollama)
    model: ""
    timeout_seconds: 30
  plugins:                   # exec contract: nonzero exit = findings
    - name: gitleaks
      exec: "gitleaks protect --staged"
      when: ["**/*"]

architecture_rules:          # your own layering assertions (Go, go/ast)
  - id: db-through-repo-layer
    description: "No direct DB calls outside internal/repo"
    language: go
    matches: "sql.Open|db.Query|db.Exec"    # regex over rendered call targets
    forbidden_outside: ["internal/repo/**"] # calls ALLOWED here, blocked elsewhere
    severity: error
```

## Modes

| Mode | Behavior |
|---|---|
| `enforce` | error-severity findings block (exit 1) — the default |
| `warn` | everything reported, exit is always 0 |
| `ci_strict` | enforce, plus `dwarpal bypass` is rejected — CI is the real wall; local hooks are DX |

## Provenance & who gets gated

`apply_gates_to: all-commits` (default) gates **every commit** — the same
rules for every author. Setting `agent-only` exempts human commits: content
gates then run only when the change is detected as agent-authored, via (in
order) the `AGENTGATE_AGENT` env var, a `Co-Authored-By` trailer matching a
configured agent identity, a configured branch prefix, or a `heuristics`
regex. Branch policy always runs (it self-no-ops for humans).

## Escape hatches (all audited)

- **Per-run rule override**: commit trailer `Dwarpal-Override: <rule-id>`
  (range/CI mode) or `DWARPAL_OVERRIDE=<rule-id>[,<rule-id>]` env (staged
  mode, where no commit message exists yet).
- **One-shot full bypass**: `dwarpal bypass --reason "..."` — arms exactly one
  commit, writes `.dwarpal/bypass.log` + a git note, rejected under
  `ci_strict`.
- **Policy-level disable**: `gates.ai_patterns.disable_rules` — visible in the
  versioned config's history.

## Environment variables

| Variable | Purpose |
|---|---|
| `AGENTGATE_AGENT` | Agent wrappers set this to self-identify |
| `DWARPAL_OVERRIDE` | Comma-separated rule IDs approved for this run |
| `DWARPAL_LLM_API_KEY` | The intent gate's provider key — never stored in config |
