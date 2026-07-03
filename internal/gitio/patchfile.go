// This file adds `dwarpal check --diff <file>` support (#4): running gates
// against a standalone unified-diff file instead of a live git repository.
// This lets a CI system (or an agent) hand Dwarpal a patch produced elsewhere
// without a git checkout being present.
package gitio

import (
	"os"
	"strings"
)

// FromPatchFile reads a unified-diff file at path and builds a *Diff from it.
//
// It reuses parseUnifiedAdded (the same added-line parser the live git path
// uses) to populate AddedLines and derive each file's Added count, so content
// gates behave identically whether the diff came from git or a patch file.
// Kind and Binary are derived here from the "---"/"+++" and "Binary files"
// headers, which parseUnifiedAdded ignores.
func FromPatchFile(path string) (*Diff, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	text := string(data)

	files := parsePatchFileHeaders(text)
	added := parseUnifiedAdded(text)
	for i := range files {
		if lines, ok := added[files[i].Path]; ok {
			files[i].AddedLines = lines
			files[i].Added = len(lines)
		}
	}
	return &Diff{Files: files}, nil
}

// parsePatchFileHeaders walks a unified-diff's file headers and returns one
// FileChange per file, in the order they appear. Each file is identified by
// either a "--- "/"+++ " header pair or, for binary files (which carry no
// hunks and so no --- /+++ pair in plain `git diff` output), a
// "Binary files a/... and b/... differ" line.
func parsePatchFileHeaders(diff string) []FileChange {
	var files []FileChange
	pendingOld, havePendingOld := "", false

	for _, line := range strings.Split(diff, "\n") {
		switch {
		case strings.HasPrefix(line, "--- "):
			pendingOld = line[len("--- "):]
			havePendingOld = true
		case strings.HasPrefix(line, "+++ "):
			newHeader := line[len("+++ "):]
			if !havePendingOld {
				continue // malformed: +++ without a preceding ---
			}
			files = append(files, FileChange{
				Path: pickPath(pendingOld, newHeader),
				Kind: kindFromHeaders(pendingOld, newHeader),
			})
			havePendingOld = false
		case strings.HasPrefix(line, "Binary files ") && strings.HasSuffix(line, " differ"):
			oldHeader, newHeader, ok := parseBinaryLine(line)
			if !ok {
				continue
			}
			files = append(files, FileChange{
				Path:   pickPath(oldHeader, newHeader),
				Kind:   kindFromHeaders(oldHeader, newHeader),
				Binary: true,
			})
		}
	}
	return files
}

// kindFromHeaders classifies a file change from its old/new diff headers:
// /dev/null on the old side means the file was added, /dev/null on the new
// side means it was deleted, otherwise it was modified.
func kindFromHeaders(oldHeader, newHeader string) ChangeKind {
	switch {
	case isDevNull(oldHeader):
		return KindAdded
	case isDevNull(newHeader):
		return KindDeleted
	default:
		return KindModified
	}
}

// pickPath returns the diff's canonical (new) path: the new header's path,
// falling back to the old header's for a deletion (whose new header is
// /dev/null).
func pickPath(oldHeader, newHeader string) string {
	if !isDevNull(newHeader) {
		return stripABPrefix(newHeader)
	}
	return stripABPrefix(oldHeader)
}

func isDevNull(header string) bool { return header == "/dev/null" }

// stripABPrefix removes the "a/" or "b/" prefix git prepends to paths in
// diff headers.
func stripABPrefix(p string) string {
	p = strings.TrimPrefix(p, "a/")
	p = strings.TrimPrefix(p, "b/")
	return p
}

// parseBinaryLine splits a "Binary files <old> and <new> differ" line into
// its old and new path headers.
func parseBinaryLine(line string) (oldHeader, newHeader string, ok bool) {
	rest := strings.TrimPrefix(line, "Binary files ")
	rest = strings.TrimSuffix(rest, " differ")
	old, new, found := strings.Cut(rest, " and ")
	if !found {
		return "", "", false
	}
	return old, new, true
}
