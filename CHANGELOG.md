# Changelog

## M0 — Walking skeleton (unreleased)

First end-to-end slice: the CLI, config, staged-diff extraction, Gate 1
(diff budget), reporting, and git hooks.

- `dwarpal init` — write starter `.dwarpal.yml` and install bypass-resistant hooks
- `dwarpal check [--json] [--range a..b]` — run the gate pipeline; exit 0/1/2
- `dwarpal hook install|uninstall` — manage hooks (chains to existing hooks)
- Gate 1 — diff budget: max lines/files/new-files with per-glob overrides
- Bypass resistance — pre-commit success marker + pre-push verification catches `--no-verify`
