# Contributing to Dwarpal

Thanks for helping build the quality firewall for AI-authored code.

## The one requirement: sign your commits (DCO)

Dwarpal uses the [Developer Certificate of Origin](DCO) (DCO), not a CLA.
There's no paperwork and no bot to authorize — you just certify that you wrote
(or have the right to submit) your change by **signing off** each commit:

```sh
git commit -s -m "your message"
```

That appends a `Signed-off-by: Your Name <you@example.com>` trailer (matching
your `git config user.name`/`user.email`). By adding it you agree to the terms
in the [DCO](DCO). Amend a missed sign-off with `git commit -s --amend`, or a
whole branch with `git rebase --signoff main`.

We chose DCO over a CLA deliberately — see
[ADR 0002](docs/decisions/0002-dco-over-cla.md).

## Dwarpal gates its own repo

This repository is guarded by Dwarpal. Before you push:

```sh
dwarpal check          # or `dwarpal agent setup <your-agent>` for the pre-flight loop
```

Your PR is also gated in CI. Expect the same rules the tool enforces
everywhere: reviewable diffs, no hardcoded secrets, no silenced lints,
tests on changed lines. If a rule is wrong, say so in the PR — false
positives are *our* bugs (`dwarpal feedback <rule> --reason "..."`).

## Development

```sh
git clone https://github.com/YellowFoxH4XOR/dwarpal
cd dwarpal
make test              # go test ./...
make build             # -> ./dwarpal
```

- New built-in rules are data, not code: add an entry (and a test) in
  `internal/gates/aipatterns/rules.go` — the community-contribution surface.
- Architecture decisions live in `docs/decisions/` (ADRs). Big calls get one.
- Specs are managed with OpenSpec (`openspec/`); substantive behavior changes
  go through a change proposal.

## Licensing

By contributing you agree your work is licensed under the project's
[Apache 2.0](LICENSE) license.
