## 1. Config schema extensions

- [ ] 1.1 Extend koanf schema with `gates.intent_check` (enabled/provider/endpoint/model/timeout) and `gates.plugins` (name/exec/when); strict unknown-key rejection (config-loading spec scenarios)
- [ ] 1.2 Compiled-in defaults: `intent_check.enabled: false`, empty `plugins` list
- [ ] 1.3 Unit tests: invalid provider, plugin missing `exec`, defaults preserved on partial overlay

## 2. gate-pipeline fail-open exception

- [ ] 2.1 Add an `Infra bool` (or equivalent) signal on gate errors so the engine can distinguish "infra error" from "verdict error" for exactly the intent gate
- [ ] 2.2 Engine: intent-gate infra error → `warn`-severity finding, does not flip exit code to 1 by itself; all other gates (including plugin) unchanged (fail closed)
- [ ] 2.3 txtar/unit tests: intent infra failure doesn't block; intent negative verdict does block; plugin exec failure still blocks

## 3. Gate 7 — Intent Verification (walking skeleton first: openai-compatible/local, no live network in tests)

- [ ] 3.1 `internal/gates/intent`: `Provider` interface `Verify(ctx, Prompt) (Verdict, error)`; fixture-based contract tests, no live network calls in CI
- [ ] 3.2 Implement `openaicompat` provider first (covers local Ollama + the interface shape); then `anthropic`, then `openai`
- [ ] 3.3 Payload builder: diff + task manifest text only, enforce token cap before send
- [ ] 3.4 Hard timeout via `context.WithTimeout` (default 30s, configurable); timeout classified as infra error
- [ ] 3.5 Wire gate into engine behind `intent_check.enabled`; skipped entirely (no network) when disabled
- [ ] 3.6 txtar tests: disabled by default, opt-in enabled, missing credentials → infra warn, negative verdict → block

## 4. Gate 8 — Plugin Gates

- [ ] 4.1 `internal/gates/plugin`: exec runner — stdin diff, `DWARPAL_DIFF_FILES` env, `when` glob filtering
- [ ] 4.2 Output handling: parse stdout as `Finding[]` JSON else nonzero-exit → single finding with output excerpt; zero-exit + unparseable → no findings
- [ ] 4.3 Fail-closed wiring: exec-launch failure (missing binary, permission) → gate infra error, blocks in enforce mode (no fail-open)
- [ ] 4.4 txtar tests: JSON output consumed, raw nonzero output, clean pass no findings, missing binary blocks

## 5. CLI additions: bypass + doctor

- [ ] 5.1 `dwarpal bypass --reason`: require non-empty reason, write git note on HEAD + append `.dwarpal/bypass.log`, exit 0
- [ ] 5.2 Reject bypass under `mode: ci_strict`: exit 1, no record written
- [ ] 5.3 `dwarpal doctor`: read-only checks — config validity, hook install status/marker, grammar availability, intent-provider reachability (when configured), plugin binary presence
- [ ] 5.4 txtar tests covering every cli-core bypass/doctor scenario

## 6. Distribution — goreleaser core

- [ ] 6.1 `.goreleaser.yaml`: darwin/linux/windows × amd64/arm64 archives, checksums, SBOM; confirm build matrix matches the tree-sitter binding strategy resolved by `spike-tree-sitter-ast` (flag if any target is blocked by an unresolved spike outcome)
- [ ] 6.2 Homebrew tap formula generation wired into the goreleaser release step
- [ ] 6.3 Docker image: `scratch`-based, multi-arch, embeds grammars; smoke test `docker run ... check --json`
- [ ] 6.4 curl install script; verify < 60s install-to-`version` on a clean container
- [ ] 6.5 `go install` path verified against a tagged release

## 7. Distribution — CI templates

- [ ] 7.1 GitHub Action wrapper (`action/`) invoking `dwarpal check --json`, emitting SARIF for PR annotation
- [ ] 7.2 GitLab CI template
- [ ] 7.3 pre-commit-framework hook definition
- [ ] 7.4 Each template documented with a copy-paste example in `docs/`

## 8. Launch collateral

- [ ] 8.1 README narrative: "why harnesses beat prompts," trust-boundary statement (no telemetry, no network unless configured), install instructions for all channels
- [ ] 8.2 3 recipe posts: Claude Code, Cursor, CI-only setups
- [ ] 8.3 Dogfood full M3 surface on the dwarpal repo itself (Gate 8 running at least one real plugin, e.g. gitleaks)

## 9. Exit criterion

- [ ] 9.1 All 8 gates enabled/dogfooded end-to-end on the dwarpal repo; `dwarpal doctor` reports healthy; verify no network call occurs with default config via a network-mocked/sandboxed test run
