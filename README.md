# Dwarpal

**Your agents write the code. Dwarpal decides what gets in.**

The name is Sanskrit/Hindi (द्वारपाल) for the guardian at a temple door — the
figure who decides who passes.

Dwarpal is an open-source, deterministic guardrail for **AI-authored** code. It
knows which changes an agent wrote — by `agent/*` branch prefix, `Co-Authored-By`
trailer, or configurable heuristic — and runs a small set of agent-specific
checks on *those* diffs, leaving human commits alone. It wires into your coding
agent's own hooks (Claude Code, Codex, OpenCode, Pi) so the agent reads *why* it
was blocked and fixes its own mistake **before a commit or PR exists**, with a
git-hook and CI backstop for enforcement.

Every finding carries a machine-readable `retry_hint` — the whole point is
in-loop self-correction, not a wall the agent hits.

## Install

**Homebrew (macOS/Linux):**

```sh
brew install --cask YellowFoxH4XOR/tap/dwarpal
```

> macOS note: the binary isn't notarized yet, but the cask **strips the
> Gatekeeper quarantine automatically on install**, so `brew` just works — no
> manual step. (Proper Apple signing is wired and dormant; see
> [docs/notarization.md](docs/notarization.md).)
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

Four checks, each an *agent* failure mode a generic linter doesn't look for —
not a re-implementation of gitleaks or semgrep:

| Gate | Catches | Default |
|---|---|---|
| `diff_budget` | Oversized, unreviewable diffs (agents love these) | 500 lines / 20 files / 10 new |
| `branch_policy` | Agent commits straight to `main`/`release/*` | error |
| `ai_patterns` | A newly added lint/type suppression, or a broadened catch that swallows the error blocking the agent | error/warn |
| `scope` | Files outside the declared task (`dwarpal task <id> --paths ...`) | error |

Gates apply to **every commit by default** — quality rules that only bind some
authors invite drift. Teams that want human commits exempt opt out with
`apply_gates_to: agent-only` (agents detected via env var, `Co-Authored-By`
trailers, `agent/*` branch prefix, or configurable heuristics). Every gate is
deterministic and fails closed — no LLM, no network, nothing to flake.

Secrets scanning, arbitrary AST assertions, coverage gates, and dead-code
ratchets are deliberately **out of scope** — gitleaks, semgrep, your coverage
tool, and your CI already own those. Dwarpal only does the part that's specific
to guarding an agent.

## Use it inside your agent

This is the primary surface. One command per tool wires Dwarpal into the agent's
own hooks so it self-corrects in-session:

```sh
dwarpal agent setup claude-code   # CLAUDE.md block + PreToolUse pre-flight hook
dwarpal agent setup codex         # AGENTS.md block
dwarpal agent setup opencode      # AGENTS.md block
dwarpal agent setup pi            # AGENTS.md block
```

The agent learns to pre-flight (`dwarpal check --explain-for-agent`), read the
`retry_hint`, and fix its own mistakes *before* committing. Claude Code
additionally gets a hook that feeds block-reasons straight back to the model.

## Enforcement backstop

Local hooks are developer experience, not security: agents can `--no-verify`.
That's why the pre-push hook verifies every pushed commit passed the gate, and
why `mode: ci_strict` + the GitHub Action are the real enforcement (and where
local override escapes carry no authority):

```yaml
- uses: YellowFoxH4XOR/dwarpal/action@v1   # SARIF annotations on the PR
```

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
- Agents: [Claude Code](docs/integrations/claude-code.md) · [Codex](docs/integrations/codex.md) · [OpenCode](docs/integrations/opencode.md) · [Pi](docs/integrations/pi.md)
- Integrations: [GitHub Actions](docs/integrations/github-actions.md) · [GitLab CI](docs/integrations/gitlab.md) · [pre-commit framework](docs/integrations/pre-commit.md) · [Docker](docs/integrations/docker.md)
- [Why harnesses beat prompts](docs/why-harnesses-beat-prompts.md) — the philosophy

## Trust promises

No telemetry, ever. No network calls — every gate is deterministic and offline.
Your diff never leaves your machine.

## Contributing

DCO, not a CLA — sign off your commits (`git commit -s`). See
[CONTRIBUTING.md](CONTRIBUTING.md).

## License

Apache 2.0 — see [LICENSE](LICENSE).
