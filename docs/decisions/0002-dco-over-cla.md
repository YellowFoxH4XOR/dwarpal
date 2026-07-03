# ADR 0002 — DCO over CLA for contributions

**Date:** 2026-07-03
**Status:** Accepted

## Context

Open-source projects gate inbound contributions with either a Contributor
License Agreement (CLA) — a signed legal document, usually enforced by a bot
tracking who has signed — or the Developer Certificate of Origin (DCO), a
per-commit `Signed-off-by` sign-off certifying provenance. PRD §11 Q5 left
this open.

## Decision

Use the **DCO**. Contributors sign off commits (`git commit -s`); no CLA, no
signing bot, no contributor database.

## Rationale

- **Lower friction.** A CLA is a documented drop-off point for first-time OSS
  contributors; the DCO is one flag on a commit. For a project whose adoption
  goal (G6) depends on external contributors, friction is the enemy.
- **No infrastructure or legal overhead.** A CLA needs a bot, a stored
  agreement, and someone to maintain both. The DCO is a text file and a git
  convention.
- **Precedent.** The Linux kernel, Git, GitLab, and most of the CNCF use the
  DCO successfully; it's well understood and trusted.

## Consequences

- Contributors must sign off (`-s`); PRs without it are asked to amend.
- No relicensing flexibility a CLA would grant (e.g. a future proprietary
  relicense) — an acceptable constraint for an Apache-2.0 project with no
  such plans.

## Revisit when

A corporate/enterprise motion needs copyright assignment or relicensing
rights that only a CLA provides.
