# Dwarpal

**Your agents write the code. Dwarpal decides what gets in.**

The name is Sanskrit/Hindi (द्वारपाल) for the guardian at a temple door — the
figure who decides who passes.

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
> Upgrading: run `brew update` first — Homebrew doesn't auto-pull taps, so
> `brew upgrade` alone may report "already installed" on a stale tap clone.

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
| `convention_drift` | Fluent-but-foreign code — naming, size, imports, error idioms | info |
| `intent` | "Does this diff do *only* what was asked?" (LLM, BYO key, fail-open) | off |
| `plugin` | Your existing tools — semgrep, gitleaks, anything with an exit code | opt-in |
| `architecture_rules` | *Your own* layering assertions (e.g. no DB calls outside `internal/repo`) | opt-in |

The `ai_patterns` and `convention_drift` rows are rule packs: `ai_patterns`
covers lint-suppressions, secrets (shape + entropy), SQL concatenation, broad
exception catches, and **near-duplicate functions** (real syntax-tree analysis
for Go/TS/TSX/Python); `convention_drift` scores added code against your repo's
own naming, function-size, import-style, and error-idiom norms.

Gates apply to **every commit by default** — quality rules that only bind
some authors invite drift. Teams that want human commits exempt opt out with
`apply_gates_to: agent-only` (agents detected via env var, `Co-Authored-By`
trailers, `agent/*` branch prefix, or configurable heuristics). Deterministic
gates fail closed; only the LLM gate fails open.

Local hooks are developer experience, not security: agents can `--no-verify`.
That's why the pre-push hook verifies every pushed commit passed the gate, and
why `mode: ci_strict` + the GitHub Action are the real enforcement:

```yaml
- uses: YellowFoxH4XOR/dwarpal/action@v1   # SARIF annotations on the PR
```

## Use it inside your agent

The gate is better as part of the agent's loop than as a wall it hits.
One command per tool:

```sh
dwarpal agent setup claude-code   # CLAUDE.md block + PreToolUse pre-flight hook
dwarpal agent setup codex         # AGENTS.md block
dwarpal agent setup opencode      # AGENTS.md block
dwarpal agent setup pi            # AGENTS.md block
```

The agent learns to pre-flight (`dwarpal check --explain-for-agent`), read
`retry_hints`, and fix its own mistakes *before* committing. Claude Code
additionally gets a hook that feeds block-reasons straight back to the model.

## Configuration

`dwarpal init` writes a commented `.dwarpal.yml` at the repo root — versioned,
so every clone shares the same policy. `dwarpal explain <rule-id>` tells you
why any rule exists and how to fix a finding. Escape hatch: `dwarpal bypass
--reason "..."` allows exactly one commit through, fully audited (log + git
note); rejected under `ci_strict`.

## Documentation

- [CLI reference](docs/cli.md) — every command and flag
- [Configuration reference](docs/configuration.md) — every `.dwarpal.yml` key
- [Rule reference](docs/rules/) — every rule: what, why, how to fix (also via `dwarpal explain`)
- [Coverage recipes](docs/recipes/coverage.md) — Go, Jest, Vitest, pytest, JaCoCo, SimpleCov, coverlet
- Agents: [Claude Code](docs/integrations/claude-code.md) · [Codex](docs/integrations/codex.md) · [OpenCode](docs/integrations/opencode.md) · [Pi](docs/integrations/pi.md)
- Integrations: [GitHub Actions](docs/integrations/github-actions.md) · [GitLab CI](docs/integrations/gitlab.md) · [pre-commit framework](docs/integrations/pre-commit.md) · [Docker](docs/integrations/docker.md)
- [Why harnesses beat prompts](docs/why-harnesses-beat-prompts.md) — the philosophy

## Trust promises

No telemetry, ever. No network calls in default operation — the only component
that can make one is the opt-in intent gate, and only to the provider you
configure. Your diff never leaves your machine otherwise.

## Contributing

DCO, not a CLA — sign off your commits (`git commit -s`). See
[CONTRIBUTING.md](CONTRIBUTING.md).

## License

Apache 2.0 — see [LICENSE](LICENSE).
