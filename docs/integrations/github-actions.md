# GitHub Actions

The published action installs dwarpal, runs `check --sarif` on the PR range,
and uploads SARIF so findings annotate the PR inline:

```yaml
name: Dwarpal
on: pull_request
permissions:
  contents: read
  security-events: write   # required for SARIF upload
jobs:
  gate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0    # the range diff needs history
      - uses: YellowFoxH4XOR/dwarpal/action@v1
        with:
          version: latest   # or a tag (v0.2.0) or commit SHA
```

Inputs: `version` (passed to `go install ...@<version>`), `mode` (defaults to
`ci_strict` — the action rewrites `.dwarpal.yml`'s mode for the run), `range`
(auto-detected from the PR SHAs when omitted), `sarif-file`,
`working-directory`. The job fails when the gate blocks; the SARIF upload
happens either way so findings are visible on the PR.

This repo dogfoods the action on its own PRs — see
[`.github/workflows/dwarpal.yml`](../../.github/workflows/dwarpal.yml).
