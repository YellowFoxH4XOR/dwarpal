package hooks

import "fmt"

// preCommitScript generates the pre-commit hook. If chain is non-empty, the
// displaced hook runs first and its failure aborts the commit. On a passing
// `dwarpal check` the staged tree hash is appended to the marker file, which
// the pre-push hook later verifies.
func preCommitScript(chain string) string {
	chainBlock := ""
	if chain != "" {
		chainBlock = fmt.Sprintf(`# Chain to the hook Dwarpal displaced at install time.
if [ -x %q ]; then
  %q "$@" || exit $?
fi

`, chain, chain)
	}
	return fmt.Sprintf(`#!/bin/sh
# Managed by Dwarpal (https://github.com/YellowFoxH4XOR/dwarpal). Do not edit.
%sif ! command -v dwarpal >/dev/null 2>&1; then
  echo "dwarpal: binary not found on PATH." >&2
  echo "  Fix your install, or remove the hooks with: dwarpal hook uninstall" >&2
  echo "  (or manually: git config --unset core.hooksPath)" >&2
  exit 1
fi

dwarpal check || exit $?

# Record a success marker keyed to the staged tree so the pre-push hook can
# tell a gated commit from one made with --no-verify.
marker="$(git rev-parse --git-dir)/dwarpal-ok"
tree="$(git write-tree)"
grep -qxF "$tree" "$marker" 2>/dev/null || echo "$tree" >> "$marker"
exit 0
`, chainBlock)
}

// prePushScript generates the pre-push hook. It refuses to push any commit
// whose tree hash is absent from the marker file — i.e. one that never passed
// the pre-commit gate (created with --no-verify).
func prePushScript(chain string) string {
	chainBlock := ""
	if chain != "" {
		chainBlock = fmt.Sprintf(`if [ -x %q ]; then
  %q "$@" || exit $?
fi

`, chain, chain)
	}
	return fmt.Sprintf(`#!/bin/sh
# Managed by Dwarpal (https://github.com/YellowFoxH4XOR/dwarpal). Do not edit.
%szero="0000000000000000000000000000000000000000"
marker="$(git rev-parse --git-dir)/dwarpal-ok"
fail=0

while read -r local_ref local_sha remote_ref remote_sha; do
  [ "$local_sha" = "$zero" ] && continue
  if [ "$remote_sha" = "$zero" ]; then
    revs="$(git rev-list "$local_sha" --not --remotes 2>/dev/null || git rev-list "$local_sha")"
  else
    revs="$(git rev-list "$remote_sha..$local_sha")"
  fi
  for sha in $revs; do
    tree="$(git rev-parse "$sha^{tree}")"
    if ! grep -qxF "$tree" "$marker" 2>/dev/null; then
      echo "dwarpal: commit $sha was not verified by the pre-commit gate" >&2
      echo "  (was it created with --no-verify?). Re-check with:" >&2
      echo "    dwarpal check --range $sha~1..$sha" >&2
      fail=1
    fi
  done
done

exit $fail
`, chainBlock)
}
