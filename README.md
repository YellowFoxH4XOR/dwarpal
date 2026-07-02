# Dwarpal

Your agents write the code. Dwarpal decides what gets in.

Dwarpal is an open-source, agent-agnostic pre-commit quality firewall for
AI-authored code. It sits between your coding agent and your repository,
running configurable gates on every staged diff before a commit lands.
It installs as a git hook so it works with any agent that drives git —
no SDK integration required.

## Status

M0 walking skeleton: CLI + Gate 1 (diff budget) + git hooks.

Only Gate 1 (diff line-count budget) is implemented today. Additional gates
are planned in later milestones.

## Quickstart

```sh
# Build
go build -o dwarpal ./cmd/dwarpal

# Install hooks and scaffold config in an existing repo
dwarpal init

# Run gates against the current staged diff
dwarpal check
```

With `make` available:

```sh
make build
./dwarpal init
./dwarpal check
```

## Configuration

`dwarpal init` writes a `.dwarpal/config.yaml` file. Edit it to tune gate
thresholds. The file is checked into the repo so every clone shares the same
policy.

## License

Apache 2.0 — see [LICENSE](LICENSE).
