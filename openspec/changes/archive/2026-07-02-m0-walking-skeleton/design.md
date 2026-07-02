## Context

Greenfield repo containing only the PRD. This change builds the M0 walking skeleton (PRD §10): the thinnest end-to-end slice — CLI → config → staged diff → one gate → report → exit code → git hook. Every architectural contract that gates 2–8 will depend on is established here. Stack decisions are already made (see openspec/config.yaml); this design records how they compose for M0 specifically.

## Goals / Non-Goals

**Goals:**
- `dwarpal init && dwarpal check` blocks an oversized staged diff in < 1s (M0 exit criterion).
- Freeze the cross-cutting contracts: `Gate` interface, `Finding` schema, exit codes, JSON output shape.
- Hook install that survives coexistence with husky/pre-commit-framework and resists agent `--no-verify` bypass.

**Non-Goals:**
- AST/tree-sitter anything (separate spike). RepoIndex is an interface stub only.
- Performance work beyond "don't be stupid" — the <1s bar is easily met by a single counting gate.
- Config migration machinery; v1 schema only.

## Decisions

**D1 — Shell out to system git; no go-git in M0.**
`internal/gitio` execs `git diff --cached --numstat -z` (counts) and `git diff --cached --unified=0` (hunks), parsing into a `Diff` model. Rationale: gh CLI's proven pattern; go-git is slower on large staged diffs and adds a heavy dependency for zero M0 benefit. Alternative considered: go-git primary (PRD §6 #4) — inverted deliberately; revisit only if a no-git environment ever matters.

**D2 — Gate interface takes an interface, not a struct, for the index.**
`Run(ctx context.Context, d *Diff, idx RepoIndex) ([]Finding, error)` where `RepoIndex` is an interface with a no-op M0 implementation. Rationale: gates 3/6 need it later; making it an interface now means the engine signature never changes. Alternative: add the parameter later — rejected, would touch every gate.

**D3 — Findings model is the single source; renderers are pure functions.**
`report.Render(w, findings, format)` with `tty` and `json` implementations (golangci-lint printers pattern). SARIF slots in at M1 without model changes. JSON schema: `{result, findings[], summary, retry_hints[]}` per PRD §5.4 — `retry_hints` populated even in M0 (diff-budget hints are the PRD's own example).

**D4 — Engine is sequential in M0.**
One gate ⇒ parallelism is speculative. The engine API (`engine.Run(ctx, cfg, diff) (Result, error)`) doesn't expose ordering, so going parallel later is internal. `stop_on_first_block: false` default honored from day one.

**D5 — Hook strategy: hooksPath directory + chain + marker.**
`dwarpal hook install` sets `core.hooksPath=.dwarpal/hooks`, writes `pre-commit` and `pre-push` scripts. Pre-commit: (a) runs any pre-existing hook it displaced (recorded at install time), (b) runs `dwarpal check`, (c) on success writes `.git/dwarpal-ok` marker keyed to the staged-tree hash. Pre-push: refuses push if HEAD's tree lacks a valid marker (catches `--no-verify` commits). Rationale: the documented Claude Code bypass (anthropics/claude-code#40117); local hooks are DX, marker+pre-push raises the bypass cost. Alternative: only pre-commit — rejected, trivially bypassed.

**D6 — Config: koanf with embedded defaults; `dwarpal init` writes a minimal file.**
Defaults compiled in; `.dwarpal.yml` overlays. Unknown keys = error (fail closed, exit 2) — catches typos like `max_line` silently doing nothing. Alternative: ignore unknown keys — rejected; a security gate whose misconfiguration is silent is worse than none.

**D7 — testscript/txtar for all acceptance tests.**
Each scenario: txtar archive sets up a fixture repo, script runs `git add` + `dwarpal check`, asserts output and exit code. Unit tests only for parsers (numstat, config). Rationale: the tool's whole contract is "repo state in → verdict out," which testscript encodes natively.

## Risks / Trade-offs

- [numstat parsing edge cases: renames, binary files, `-` counts] → treat binary as 0 lines / 1 file; resolve renames via `--find-renames` status letters; txtar fixtures for each case.
- [hooksPath collides with husky (which also sets core.hooksPath)] → detect existing hooksPath at install, record it, chain to it; refuse with a clear message if chaining is impossible rather than clobber.
- [marker file gives false security if user never installs pre-push] → `dwarpal doctor` (M1) reports hook health; README states plainly that ci_strict is the real enforcement.
- [Windows hook scripts (sh shebang) unverified] → known gap, PRD §11 Q3; CI matrix later, not an M0 gate.

## Open Questions

- Module path: `github.com/YellowFoxH4XOR/dwarpal` vs `github.com/dwarpal/dwarpal` (org is reserved per PRD §11 Q1). Needs user confirmation before `go mod init`.
