# Dwarpal

**Your agents write the code. Dwarpal decides what gets in.**

Dwarpal is an open-source, agent-agnostic pre-commit quality firewall for
AI-authored code. It sits between your coding agent and your repository,
running deterministic gates on every staged diff before a commit lands. It
installs as a git hook, so it works with any agent that drives git — Claude
Code, Cursor, Aider, Devin, Copilot — no SDK integration required.

Blocked commits read: **"Dwarpal stopped this at the gate."** And every block
carries machine-readable `retry_hints`, so your agent can read *why* it was
blocked and fix its own mistake.

## Install

**Homebrew (macOS/Linux):**

```sh
brew install --cask YellowFoxH4XOR/tap/dwarpal
```

> macOS note: the binary is not yet notarized. If Gatekeeper blocks it, run
> `xattr -d com.apple.quarantine "$(readlink -f /opt/homebrew/bin/dwarpal)"`.

**Install script** (handles the quarantine step automatically):

```sh
curl -fsSL https://raw.githubusercontent.com/YellowFoxH4XOR/dwarpal/main/install.sh | sh
```

**Go:**

```sh
go install github.com/YellowFoxH4XOR/dwarpal/cmd/dwarpal@latest
```

## Quickstart

```sh
cd your-repo
dwarpal init      # writes .dwarpal.yml, installs pre-commit + pre-push hooks
dwarpal check     # runs the gate pipeline against staged changes
dwarpal rules     # shows every active gate and rule
```

## The gates

| Gate | Catches | Default |
|---|---|---|
| `diff_budget` | Oversized, unreviewable diffs | 500 lines / 20 files / 10 new |
| `branch_policy` | Agent commits straight to `main`/`release/*` | error |
| `ai_patterns` | Lint suppressions, hardcoded secrets, string-built SQL, broad catches, near-duplicate functions | error/warn |
| `scope` | Files outside the declared task (`dwarpal task <id> --paths ...`) | error |
| `diff_coverage` | Under-tested changed lines (lcov/cobertura/go-cover) | opt-in |
| `convention_drift` | Fluent-but-foreign code that bucks repo style | info |
| `intent` | "Does this diff do *only* what was asked?" (LLM, BYO key, fail-open) | off |
| `plugin` | Your existing tools — semgrep, gitleaks, anything with an exit code | opt-in |

Gates apply to **agent-authored commits only** by default (detected via env
var, `Co-Authored-By` trailers, or `agent/*` branch prefix) — human commits
stay untouched. Deterministic gates fail closed; only the LLM gate fails open.

Local hooks are developer experience, not security: agents can `--no-verify`.
That's why the pre-push hook verifies every pushed commit passed the gate, and
why `mode: ci_strict` + the GitHub Action are the real enforcement:

```yaml
- uses: YellowFoxH4XOR/dwarpal/action@v1   # SARIF annotations on the PR
```

## Configuration

`dwarpal init` writes a commented `.dwarpal.yml` at the repo root — versioned,
so every clone shares the same policy. `dwarpal explain <rule-id>` tells you
why any rule exists and how to fix a finding. Escape hatch: `dwarpal bypass
--reason "..."` allows exactly one commit through, fully audited (log + git
note); rejected under `ci_strict`.

## Trust promises

No telemetry, ever. No network calls in default operation — the only component
that can make one is the opt-in intent gate, and only to the provider you
configure. Your diff never leaves your machine otherwise.

## License

Apache 2.0 — see [LICENSE](LICENSE).
