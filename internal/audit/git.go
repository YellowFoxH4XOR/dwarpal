package audit

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/YellowFoxH4XOR/dwarpal/internal/gitio"
)

// recentCommits returns the SHAs of the most recent non-merge commits, newest
// first, capped at window. We deliberately do NOT pass --first-parent: on a
// PR-merge workflow the mainline is almost all merge commits, so first-parent +
// no-merges would starve the sample down to the few direct-to-main commits. The
// -n cap already bounds the replay, so walking all reachable non-merge commits
// (including the PR-branch commits where edits actually happened) is both safe
// and what we want to sample.
func recentCommits(root string, window int) ([]string, error) {
	out, err := gitOut(root, "log", "--no-merges",
		"-n", strconv.Itoa(window), "--format=%H")
	if err != nil {
		return nil, err
	}
	var shas []string
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if line != "" {
			shas = append(shas, line)
		}
	}
	return shas, nil
}

// materialize writes commit c's version of each file touched by diff d into
// dir, preserving relative paths, so the AST tier can parse historical content.
// Deleted files (no blob at c) are skipped — they produce no added lines anyway.
func materialize(root, c string, d *gitio.Diff, dir string) error {
	for _, f := range d.Files {
		if len(f.AddedLines) == 0 {
			continue // nothing for a content gate to inspect
		}
		blob, err := gitOut(root, "show", c+":"+f.Path)
		if err != nil {
			continue // file absent at c (e.g. pure rename edge) — skip, not fatal
		}
		dst := filepath.Join(dir, f.Path)
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(dst, []byte(blob), 0o644); err != nil {
			return err
		}
	}
	return nil
}

// addedLineText maps each added line to its text, keyed by file+line number, so
// a finding (which carries only file+line) can recover the flagged line's
// content for HEAD resolution.
func addedLineText(d *gitio.Diff) map[string]string {
	m := map[string]string{}
	for _, f := range d.Files {
		for _, ln := range f.AddedLines {
			m[lineKey(f.Path, ln.Number)] = ln.Text
		}
	}
	return m
}

func lineKey(file string, line int) string { return file + ":" + strconv.Itoa(line) }

// headBlob returns HEAD's content for path, or nil if the path does not exist
// at HEAD (removed or renamed away).
func headBlob(root, path string) *string {
	out, err := gitOut(root, "show", "HEAD:"+path)
	if err != nil {
		return nil
	}
	return &out
}

func round2(f float64) float64 { return float64(int(f*100+0.5)) / 100 }

func gitOut(root string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	out, err := cmd.Output()
	return string(out), err
}
