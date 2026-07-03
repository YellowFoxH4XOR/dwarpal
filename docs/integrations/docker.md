# Docker

The image bundles dwarpal + git (required — dwarpal shells out to git) and
trusts mounted repos (`safe.directory *`, standard for single-purpose CI
containers where the host uid differs from the container's).

Build locally (image publishing to a registry is on the roadmap):

```sh
docker build -t dwarpal https://github.com/YellowFoxH4XOR/dwarpal.git
```

Run against a repo:

```sh
docker run --rm -v "$PWD:/repo" -w /repo dwarpal check
docker run --rm -v "$PWD:/repo" -w /repo dwarpal check --range origin/main..HEAD --json
```

Exit codes pass through: `0` pass, `1` blocked, `2` config/internal error —
so `docker run ... check` works directly as a CI step.
