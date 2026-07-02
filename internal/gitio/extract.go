package gitio

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// ErrGitNotFound is returned when the system git binary is unavailable. The CLI
// maps this to exit code 2 with a message that system git is required.
var ErrGitNotFound = errors.New("system git executable not found on PATH")

// Extractor pulls a Diff out of a git repository. dir is the working directory
// git runs in (the repo root or any path inside it); empty means the process
// CWD. Keeping dir a field makes the extractor trivial to point at a throwaway
// fixture repo in tests.
type Extractor struct {
	dir string
}

// NewExtractor builds an Extractor rooted at dir.
func NewExtractor(dir string) *Extractor { return &Extractor{dir: dir} }

// Staged returns the diff of the staging area (index vs HEAD), the default
// target for a pre-commit gate.
func (e *Extractor) Staged() (*Diff, error) {
	return e.diff([]string{"--cached"})
}

// Range returns the diff between two revisions, e.g. "HEAD~1..HEAD".
func (e *Extractor) Range(spec string) (*Diff, error) {
	return e.diff([]string{spec})
}

// diff runs numstat and name-status with the given selector args and merges
// them into a single Diff. numstat supplies line counts and binary detection;
// name-status supplies the change kind. Both are keyed by the (new) path.
func (e *Extractor) diff(selector []string) (*Diff, error) {
	counts, err := e.run(append([]string{"diff", "--numstat", "-z", "--find-renames"}, selector...))
	if err != nil {
		return nil, err
	}
	status, err := e.run(append([]string{"diff", "--name-status", "-z", "--find-renames"}, selector...))
	if err != nil {
		return nil, err
	}

	kinds := parseNameStatus(status)
	files := parseNumstat(counts)
	for i := range files {
		if k, ok := kinds[files[i].Path]; ok {
			files[i].Kind = k
		}
	}

	// Enrich with added-line content so content gates (secrets, suppressions,
	// AST rules) can inspect the actual changed code, not just counts. A
	// failure here is non-fatal: counts-only gates still work.
	if added, err := e.addedContent(selector); err == nil {
		for i := range files {
			files[i].AddedLines = added[files[i].Path]
		}
	}

	return &Diff{Files: files}, nil
}

// run executes git with args and returns stdout. A missing git binary maps to
// ErrGitNotFound; any other failure surfaces git's stderr for diagnosis.
func (e *Extractor) run(args []string) ([]byte, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = e.dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return nil, ErrGitNotFound
		}
		return nil, fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, stderr.String())
	}
	return stdout.Bytes(), nil
}

// parseNumstat decodes `git diff --numstat -z --find-renames`.
//
// Each record is `<added>\t<removed>\t` followed by the path info. For a normal
// file the path runs until the next NUL. For a rename, the path field is empty
// (an immediate NUL) and the two following NUL-terminated tokens are the old
// and new paths. Binary files report "-" for both counts.
func parseNumstat(b []byte) []FileChange {
	tokens := splitNUL(b)
	var files []FileChange
	i := 0
	for i < len(tokens) {
		head := tokens[i]
		i++
		// head is "<added>\t<removed>\t<maybe-path>" (path present for normal files).
		parts := strings.SplitN(head, "\t", 3)
		if len(parts) < 3 {
			continue // malformed record; skip defensively
		}
		fc := FileChange{Kind: KindModified}
		if parts[0] == "-" || parts[1] == "-" {
			fc.Binary = true
		} else {
			fc.Added, _ = strconv.Atoi(parts[0])
			fc.Removed, _ = strconv.Atoi(parts[1])
		}
		if parts[2] == "" {
			// Rename: the next two tokens are old and new paths.
			if i+1 < len(tokens) {
				fc.OldPath = tokens[i]
				fc.Path = tokens[i+1]
				fc.Kind = KindRenamed
				i += 2
			}
		} else {
			fc.Path = parts[2]
		}
		files = append(files, fc)
	}
	return files
}

// parseNameStatus decodes `git diff --name-status -z --find-renames` into a
// map from (new) path to ChangeKind. Records are `<status>\0<path>\0`, except
// renames/copies which are `R<score>\0<oldpath>\0<newpath>\0`.
func parseNameStatus(b []byte) map[string]ChangeKind {
	tokens := splitNUL(b)
	kinds := map[string]ChangeKind{}
	i := 0
	for i < len(tokens) {
		status := tokens[i]
		i++
		if status == "" {
			continue
		}
		switch status[0] {
		case 'A':
			if i < len(tokens) {
				kinds[tokens[i]] = KindAdded
				i++
			}
		case 'D':
			if i < len(tokens) {
				kinds[tokens[i]] = KindDeleted
				i++
			}
		case 'R', 'C':
			// R<score>\0old\0new — key by the new path.
			if i+1 < len(tokens) {
				kinds[tokens[i+1]] = KindRenamed
				i += 2
			}
		default: // M, T, etc.
			if i < len(tokens) {
				kinds[tokens[i]] = KindModified
				i++
			}
		}
	}
	return kinds
}

// splitNUL splits a NUL-delimited byte stream into string tokens, dropping the
// trailing empty token that a NUL terminator leaves behind.
func splitNUL(b []byte) []string {
	if len(b) == 0 {
		return nil
	}
	raw := strings.Split(string(b), "\x00")
	out := make([]string, 0, len(raw))
	for _, t := range raw {
		if t == "" {
			continue
		}
		out = append(out, t)
	}
	return out
}
