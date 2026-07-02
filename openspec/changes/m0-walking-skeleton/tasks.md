## 1. Repo scaffold

- [x] 1.1 Confirm module path with owner, then `go mod init`; add LICENSE (Apache 2.0), .gitignore, minimal README stub
- [x] 1.2 Create package layout: `cmd/dwarpal`, `internal/{config,gitio,engine,gates/diffbudget,report,hooks}`
- [x] 1.3 Add deps: spf13/cobra, knadh/koanf, rogpeppe/go-internal (test); wire a `make build test` (or Taskfile) target
- [x] 1.4 Set up testscript harness: one trivial txtar test proving `dwarpal version` runs end-to-end

## 2. Core contracts (freeze these first)

- [x] 2.1 Define `Finding` struct and severity enum in a shared package (fields per gate-pipeline spec)
- [x] 2.2 Define `Gate` interface `{ID() string; Run(ctx, *Diff, RepoIndex) ([]Finding, error)}` with no-op `RepoIndex` interface stub
- [x] 2.3 Define `Diff` model: per-file path, kind (add/modify/delete/rename), added/removed counts, binary flag

## 3. Config loading

- [x] 3.1 Compiled-in defaults (enforce mode; 500/20/10 budgets); koanf overlay from `.dwarpal.yml` at repo root
- [x] 3.2 Strict validation: unknown keys and out-of-domain values → exit 2 naming the key (config-loading spec scenarios)
- [x] 3.3 Unit tests: partial overlay, typo key, invalid mode

## 4. Diff extraction (gitio)

- [x] 4.1 Locate git binary and repo root; exit 2 with clear message when missing (diff-extraction spec)
- [x] 4.2 Parse `git diff --cached --numstat -z --find-renames` into the Diff model; handle binary (`-`), renames, spaces/non-ASCII paths
- [x] 4.3 `--range <a>..<b>` mode using the same parser
- [x] 4.4 txtar tests: modified+new, empty staging, binary file, rename

## 5. Engine + Gate 1

- [x] 5.1 Engine: run enabled gates in order, aggregate findings, honor `stop_on_first_block: false` default; gate infra error → blocked (fail closed)
- [x] 5.2 Diff-budget gate: max_lines/max_files/max_new_files, one finding per exceeded budget, severity error
- [x] 5.3 Per-glob overrides: first matching override wins; files grouped per budget (gate-diff-budget mixed-diff scenario)
- [x] 5.4 Retry hints: imperative message with actual vs allowed counts on every budget finding

## 6. Report + CLI wiring

- [x] 6.1 TTY renderer: findings grouped by gate, file:line, suggestion; summary line
- [x] 6.2 JSON encoder: `{result, findings[], summary, retry_hints[]}`; diagnostics to stderr only in --json mode
- [x] 6.3 `dwarpal check` command: staged default, `--range`, `--json`; exit codes 0/1/2; `warn` mode exits 0 with findings
- [x] 6.4 `dwarpal version` with build-time ldflags
- [x] 6.5 txtar acceptance tests covering every cli-core spec scenario

## 7. Hooks

- [x] 7.1 `dwarpal hook install`: create `.dwarpal/hooks/`, set core.hooksPath, detect + record displaced hooks/hooksPath, refuse-not-clobber when chaining impossible
- [x] 7.2 pre-commit script: chain displaced hook → `dwarpal check` → write staged-tree-hash marker in `.git/` on pass
- [x] 7.3 pre-push script: verify marker for pushed commits; block and name unverified commits (`--no-verify` catch)
- [x] 7.4 `dwarpal hook uninstall`: restore prior hooksPath/hooks state
- [x] 7.5 Missing-binary hook message includes uninstall instructions
- [x] 7.6 txtar tests: clean install, husky coexistence, no-verify caught at push, uninstall restore

## 8. init + M0 exit criterion

- [x] 8.1 `dwarpal init`: git-repo check, write starter `.dwarpal.yml` (never overwrite), install hooks, print actions
- [x] 8.2 End-to-end acceptance: fixture repo, stage 600-line diff, `dwarpal init && dwarpal check` blocks; time the run and assert < 1s
- [ ] 8.3 Dogfood: run `dwarpal init` on the dwarpal repo itself; all subsequent commits to this repo pass through the gate
