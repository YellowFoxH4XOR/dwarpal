# GitLab CI

Copy-paste template — runs the gate on merge requests using the Docker image
(which bundles git) or the install script:

```yaml
# .gitlab-ci.yml
dwarpal:
  stage: test
  image: golang:1.26            # or your own image; needs git
  rules:
    - if: $CI_PIPELINE_SOURCE == "merge_request_event"
  variables:
    GIT_DEPTH: "0"              # full history for the range diff
  script:
    - go install github.com/YellowFoxH4XOR/dwarpal/cmd/dwarpal@latest
    - export PATH="$PATH:$(go env GOPATH)/bin"
    # gate the MR's commits; --json for the machine-readable verdict artifact
    - dwarpal check --range "origin/$CI_MERGE_REQUEST_TARGET_BRANCH_NAME..HEAD" --json | tee dwarpal.json
  artifacts:
    when: always
    paths: [dwarpal.json]
```

Notes:
- Exit codes are the contract: the job fails (exit 1) when the gate blocks,
  and errors (exit 2) on config problems — no output parsing needed.
- For `ci_strict` enforcement, set `mode: ci_strict` in the repo's
  `.dwarpal.yml` (versioned — the MR itself can't weaken it without the
  change being visible in the diff).
- GitLab's code-quality widget accepts a different JSON schema than SARIF;
  until a native converter exists, the `dwarpal.json` artifact + job status is
  the integration surface.
