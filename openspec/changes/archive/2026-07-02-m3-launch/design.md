## Context

M0–M2 shipped six deterministic, no-network gates plus the CLI/config/hook
scaffolding. M3 closes the PRD's gate set (§5.2 Gates 7 & 8) and adds
distribution (§5.5) so Dwarpal can actually be installed and launched. This
is the first change that touches the network at all — Gate 7 is BYO-key LLM
intent verification — and the first that ships an `exec` contract letting
users run arbitrary external commands as gates. Both widen Dwarpal's trust
surface, so design here is mostly about containing that: default-off,
explicit opt-in, hard timeouts, and one narrowly-scoped fail-open exception
to the fail-closed rule established in M0/M1 (`gate-pipeline` "Deterministic
gates fail closed").

Distribution (goreleaser/Homebrew/Docker/CI) has no gate-pipeline
dependency and can be built in parallel with Gates 7/8.

## Goals / Non-Goals

**Goals:**
- Gate 7 (intent) ships **off by default**, BYO key, and never blocks a
  commit on provider/infra failure — the one documented exception to
  fail-closed (PRD §6 #6).
- Gate 8 (plugin) turns any exec-able tool into a Dwarpal gate via a small,
  stable contract (stdin diff, JSON-or-raw stdout, exit code = verdict).
- `dwarpal init && dwarpal check` installable and runnable in < 60s (PRD
  G2) via Homebrew, `go install`, curl script, or Docker.
- `dwarpal bypass --reason` and `dwarpal doctor` close the M0 B5/B7
  blockers (hook health visibility, auditable bypass).
- Trust promises stay literally true: no telemetry, no network unless the
  user configures Gate 7 or a network-calling plugin.

**Non-Goals:**
- Hosted key management, hosted intent verification (Cloud tier, Phase 2).
- A plugin marketplace/registry — v1 is "point `exec` at a local binary."
- Sandboxing plugin execution (out of scope; documented as the user's
  responsibility, same trust model as a Makefile target).
- Windows-signed installers / MSI — goreleaser's default archive + curl
  script only for v1.

## Decisions

**D1 — Gate 7 fail-open is scoped to *infrastructure* errors only, never to
verdicts.** `internal/gates/intent` distinguishes (a) transport/timeout/
auth/rate-limit errors → `warn`-severity finding, exit code unaffected by
this gate, and (b) a successful LLM response with a "does not match intent"
verdict → normal `error`-severity finding that blocks like any other gate.
Rationale: PRD §6 #6 says "infra never blocks," not "the gate is toothless."
Alternative considered: fail-open on any non-pass result — rejected, that
would make the gate decorative once a team turns it on.

**D2 — Gate 7 provider abstraction is a minimal interface, not a full SDK
wrapper.** `type Provider interface { Verify(ctx, Prompt) (Verdict, error) }`
with three implementations (`anthropic`, `openai`, `openaicompat` — the last
also serves local Ollama since it speaks the OpenAI chat-completions shape).
Rationale: PRD lists exactly these three; a generic interface avoids
vendoring a provider SDK per option and keeps the binary dependency-light.
Hard timeout (default 30s, configurable) and token cap enforced by the
caller via `context.WithTimeout`, not left to provider clients.

**D3 — Gate 7 sends only the unified diff + task manifest text, never repo
contents beyond the diff.** No RepoIndex access from this gate. Rationale:
the README's trust-boundary claim ("diff never leaves the machine unless
configured") must hold literally — bounding the payload to what the user
already opted to send is the whole point of the boundary.

**D4 — Gate 8 exec contract: diff on stdin, JSON findings on stdout if
parseable, else nonzero exit = one synthetic finding.** Command receives the
unified diff on stdin and `DWARPAL_DIFF_FILES` (newline-separated changed
paths) in env; `when` globs in config filter which files trigger the plugin.
If stdout parses as the `Finding[]` JSON shape, findings are used directly
(gate ID rewritten to `plugin:<name>`); otherwise a nonzero exit produces one
finding with the raw stdout/stderr tail as the message. Rationale: matches
gitleaks/semgrep's existing `--json` output habits while still being useful
against tools with no structured output (mirrors golangci-lint's exec-linter
adapters). Alternative: require strict JSON schema from every plugin —
rejected, would exclude most real-world CLIs teams already run.

**D5 — Gate 8 plugins are deterministic and fail closed**, same as Gates
1–6: a plugin that errors to run (binary missing, nonzero from a shell
misconfiguration distinguishable from tool findings via a documented
`when: exit_code` convention) blocks in `enforce` mode. Only Gate 7 gets the
infra fail-open exception (D1). Rationale: an exec gate wrapping semgrep
should have the same trust guarantee as a built-in AST rule; the fail-open
carve-out is specific to third-party LLM availability, not to "external
process" in general.

**D6 — Distribution: goreleaser is the single source of release artifacts.**
One `.goreleaser.yaml` drives GitHub Releases (darwin/linux/windows ×
amd64/arm64 archives + checksums + SBOM), the Homebrew tap formula, and the
Docker image (`scratch`-based, static binary, multi-arch manifest).
`go install` and the curl script both consume the same GitHub Release
artifacts — no separate build path to keep in sync. Rationale: goreleaser is
already the PRD's chosen tool (config.yaml) and is the gitleaks-style
exemplar; a single pipeline avoids version drift between install channels.

**D7 — `dwarpal doctor` is read-only diagnostics, no auto-fix.** Reports
config validity, hook install state (hooksPath, marker presence), grammar
availability, and Gate 7/8 reachability (does the configured provider/binary
exist) without mutating anything. Rationale: matches `cli-core`'s existing
"init never overwrites" caution; a diagnostics command that silently repairs
state is surprising and harder to reason about in CI.

**D8 — `dwarpal bypass --reason` writes to both a git note on HEAD and a
local append-only log (`.dwarpal/bypass.log`), and is rejected outright
under `mode: ci_strict`.** Rationale: PRD §5.1 requires the bypass be
auditable, and R7 (bypass culture) is mitigated specifically by `ci_strict`
being the real enforcement — allowing `bypass` there would defeat that
design. The git note keeps the record attached to the commit even if the
local log is lost/gitignored by mistake.

## Risks / Trade-offs

- [Gate 7 network calls are the first crack in "no network in default
  operation"] → off by default, requires explicit `intent_check.enabled:
  true` + provider config; `doctor` and docs make the opt-in loud; diff-only
  payload (D3) bounds exposure.
- [Gate 8 lets a config file execute arbitrary commands — a supply-chain
  risk if `.dwarpal.yml` is attacker-controlled] → document plainly that
  `plugins:` entries execute with the user's local permissions (same trust
  model as `package.json` scripts / Makefiles); no sandboxing in v1;
  `dwarpal doctor` lists configured plugin commands so they're reviewable.
- [Provider API shape drift (Anthropic/OpenAI change response schema)] →
  provider clients isolated behind the D2 interface; contract-tested against
  fixture responses, not live calls, in CI.
- [goreleaser Homebrew tap requires a second repo (`homebrew-tap`) with its
  own push permissions] → scope to a single maintainer-owned tap repo;
  document the token requirement in `design.md`/release runbook, not
  end-user facing.
- [Cross-compiling with tree-sitter grammars onto the windows/arm64 and
  other matrix cells may hit the same cgo-vs-WASM concerns as the spike] →
  distribution build matrix reuses whatever binding strategy
  `spike-tree-sitter-ast` already resolved for M1/M2; no new decision here,
  just confirm the goreleaser matrix matches what M1 already ships.

## Open Questions

- Exact Gate 7 prompt template and structured-verdict schema (surprise list
  format) — left to implementation; not a spec-blocking decision since the
  scenario only requires a pass/fail verdict + message, per PRD §5.2.
- Whether the Homebrew tap is `dwarpal/tap` under the reserved GitHub org
  (PRD §11 Q1) — depends on the module-path question already open since M0;
  does not block writing this change's specs.
