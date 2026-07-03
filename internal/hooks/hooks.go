// Package hooks installs and removes Dwarpal's git hooks.
//
// Strategy (design decision D5):
//   - Set core.hooksPath to a Dwarpal-managed directory so the hooks are
//     versioned-in-repo-adjacent and every clone can re-install identically.
//   - Chain to any hooks we displace (a prior core.hooksPath or .git/hooks)
//     rather than clobbering them, so Dwarpal coexists with husky et al.
//   - Bypass resistance: the pre-commit hook writes a success marker keyed to
//     the staged tree hash; the pre-push hook refuses to push commits whose
//     tree lacks a marker. This catches `git commit --no-verify`, the
//     documented agent bypass (anthropics/claude-code#40117). Local hooks are
//     DX; ci_strict remains the real enforcement.
package hooks

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// dirRel is the Dwarpal hooks directory, relative to the repo root.
const dirRel = ".dwarpal/hooks"

// prevKey stores the pre-install core.hooksPath so uninstall can restore it.
const prevKey = "dwarpal.previoushookspath"

// Install writes the hooks and points git at them. root is the repo work tree.
func Install(root string) ([]string, error) {
	var actions []string

	hooksDir := filepath.Join(root, dirRel)
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		return nil, err
	}

	prev, _ := gitOut(root, "config", "--local", "--get", "core.hooksPath")
	prev = strings.TrimSpace(prev)

	// Resolve the hooks we would displace, so the scripts can chain to them.
	chainPreCommit := existingHook(root, prev, "pre-commit", hooksDir)
	chainPrePush := existingHook(root, prev, "pre-push", hooksDir)

	if err := writeScript(filepath.Join(hooksDir, "pre-commit"), preCommitScript(chainPreCommit)); err != nil {
		return nil, err
	}
	if err := writeScript(filepath.Join(hooksDir, "pre-push"), prePushScript(chainPrePush)); err != nil {
		return nil, err
	}
	actions = append(actions, "wrote pre-commit and pre-push hooks to "+dirRel)
	if chainPreCommit != "" {
		actions = append(actions, "chaining to existing pre-commit: "+chainPreCommit)
	}

	// Remember the prior hooksPath (only on first install) for a clean restore.
	if prev != "" && prev != dirRel && prev != hooksDir {
		if existing, _ := gitOut(root, "config", "--local", "--get", prevKey); strings.TrimSpace(existing) == "" {
			_ = gitRun(root, "config", "--local", prevKey, prev)
		}
	}

	if err := gitRun(root, "config", "--local", "core.hooksPath", dirRel); err != nil {
		return nil, fmt.Errorf("setting core.hooksPath: %w", err)
	}
	actions = append(actions, "set core.hooksPath to "+dirRel)
	return actions, nil
}

// Uninstall restores the pre-install hook configuration.
func Uninstall(root string) ([]string, error) {
	var actions []string
	prev, _ := gitOut(root, "config", "--local", "--get", prevKey)
	prev = strings.TrimSpace(prev)
	if prev != "" {
		if err := gitRun(root, "config", "--local", "core.hooksPath", prev); err != nil {
			return nil, err
		}
		_ = gitRun(root, "config", "--local", "--unset", prevKey)
		actions = append(actions, "restored core.hooksPath to "+prev)
	} else {
		_ = gitRun(root, "config", "--local", "--unset", "core.hooksPath")
		actions = append(actions, "unset core.hooksPath")
	}
	return actions, nil
}

// existingHook returns the path of a hook Dwarpal would displace, if any: first
// a prior core.hooksPath's hook, else a .git/hooks hook. It never returns a
// hook already inside our own directory (avoids self-chaining loops).
func existingHook(root, prevHooksPath, name, ownDir string) string {
	candidates := []string{}
	if prevHooksPath != "" {
		candidates = append(candidates, resolve(root, prevHooksPath, name))
	}
	gitDir, err := gitOut(root, "rev-parse", "--git-dir")
	if err == nil {
		candidates = append(candidates, resolve(root, strings.TrimSpace(gitDir), filepath.Join("hooks", name)))
	}
	for _, c := range candidates {
		if c == "" || strings.HasPrefix(c, ownDir) {
			continue
		}
		if isExecutableFile(c) {
			return c
		}
	}
	return ""
}

func resolve(root, base, name string) string {
	p := filepath.Join(base, name)
	if !filepath.IsAbs(p) {
		p = filepath.Join(root, p)
	}
	return p
}

func isExecutableFile(p string) bool {
	fi, err := os.Stat(p)
	if err != nil || fi.IsDir() {
		return false
	}
	// Windows/NTFS has no unix exec bit, and Git for Windows runs hooks by
	// shebang regardless — so any regular file in a hooks dir is runnable.
	// On unix, git only runs +x hooks, so we chain only to those.
	if runtime.GOOS == "windows" {
		return true
	}
	return fi.Mode()&0o111 != 0
}

func writeScript(path, body string) error {
	return os.WriteFile(path, []byte(body), 0o755)
}

func gitRun(root string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, stderr.String())
	}
	return nil
}

func gitOut(root string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	out, err := cmd.Output()
	return string(out), err
}
