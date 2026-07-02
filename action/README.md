# dwarpal GitHub Action

Runs the `dwarpal` quality firewall against a pull request or push, and
uploads the results as SARIF so findings show up as PR annotations.

## Usage

```yaml
name: dwarpal

on:
  pull_request:
  push:
    branches: [main]

permissions:
  contents: read
  security-events: write

jobs:
  dwarpal:
    runs-on: ubuntu-latest
    steps:
      - uses: YellowFoxH4XOR/dwarpal/action@v1
```

`fetch-depth: 0` is handled for you (the Action checks out the repo itself),
so you don't need a separate `actions/checkout` step.

## Inputs

| Input                | Description                                                                 | Default          |
| --------------------- | ---------------------------------------------------------------------------- | ----------------- |
| `version`             | dwarpal version to install (`go install .../dwarpal@<version>`), or `latest` | `latest`          |
| `mode`                | Gate mode to enforce for this run: `enforce`, `warn`, or `ci_strict`         | `ci_strict`        |
| `range`               | Commit range for `dwarpal check --range`. Auto-detected for PRs/pushes if omitted | `""` (auto-detect) |
| `working-directory`   | Directory to run `dwarpal` in                                                | `.`               |
| `sarif-file`          | Path to write the SARIF report to                                            | `dwarpal.sarif`   |

## Outputs

| Output       | Description                                  |
| ------------- | --------------------------------------------- |
| `sarif-file`  | Path to the generated SARIF report            |
| `conclusion`  | `success` or `failure`, reflecting the `dwarpal check` exit code |

## Notes

- `security-events: write` permission is required on the calling workflow for
  the SARIF upload step (`github/codeql-action/upload-sarif`) to succeed.
- `mode` defaults to `ci_strict` so CI is never softer than local hooks, even
  if the repo's `.dwarpal.yml` sets `mode: warn`.
- Pin to a major version tag (`@v1`) for compatibility; avoid `@main`.
