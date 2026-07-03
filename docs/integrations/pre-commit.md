# pre-commit framework

For teams already standardized on [pre-commit](https://pre-commit.com), this
repo ships a hook definition — add to your `.pre-commit-config.yaml`:

```yaml
repos:
  - repo: https://github.com/YellowFoxH4XOR/dwarpal
    rev: v0.2.0            # pin a release tag
    hooks:
      - id: dwarpal
```

This runs `dwarpal check` against the staged diff on every commit, building
dwarpal from source via Go (pre-commit's `golang` language support handles the
toolchain).

**Trade-off vs `dwarpal init`'s native hooks**: the pre-commit framework
manages only the pre-commit stage by default, so you don't get Dwarpal's
pre-push marker verification (the `--no-verify` catch) unless you also enable
the hook for the `pre-push` stage:

```yaml
      - id: dwarpal
        stages: [pre-commit, pre-push]
```

Native `dwarpal init` hooks remain the fuller experience (marker plumbing,
one-shot bypass consumption); the framework definition exists so adopting
Dwarpal doesn't force a hooks-manager migration.


## Windows

Dwarpal's hooks are POSIX `sh` scripts; **Git for Windows runs them via its
bundled bash**, so `dwarpal init` and the pre-commit/pre-push flow work under
Windows git (CI is verified on `windows-latest`). Two Windows specifics:
- Hook chaining detects displaced hooks by *existence* (NTFS has no exec bit),
  matching how Git for Windows runs hooks by shebang.
- Use Git Bash or a git client backed by Git for Windows; a POSIX `sh` on
  `PATH` is required for the hooks to execute.
