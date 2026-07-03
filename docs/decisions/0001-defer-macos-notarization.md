# ADR 0001 — Defer macOS notarization

**Date:** 2026-07-03
**Status:** Accepted

## Context

Unsigned macOS binaries are quarantined by Gatekeeper and killed on first run.
The proper fix is Apple code signing + notarization; GoReleaser's built-in
quill makes this cross-platform (no macOS runner). It requires a paid Apple
Developer account ($99/yr) and ongoing management of a Developer ID
certificate and an App Store Connect API key.

## Decision

Wire the notarization pipeline but leave it **dormant** (guarded by
`isEnvSet MACOS_SIGN_P12`), and **do not activate it** at this stage. Users on
macOS rely on the interim mitigations:

- the `curl | sh` install script strips `com.apple.quarantine` automatically;
- the README documents the one-line `xattr -d` fix for the Homebrew cask.

## Consequences

- No recurring cost or credential-rotation burden while the project is early.
- macOS users see a Gatekeeper prompt (cask) unless they run the documented
  one-liner — accepted friction at this stage.
- Activation is purely additive later: set five repo secrets, tag a release
  (docs/notarization.md). No code change, no rework — the config stays
  validated by `goreleaser check` on every release, so it can't rot.

## Revisit when

Adoption grows enough that the Gatekeeper prompt is a real install-funnel
drop-off, or an Apple Developer account exists for other reasons.
