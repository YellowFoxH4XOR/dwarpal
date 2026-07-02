// Package gitio extracts the change under inspection from git.
//
// Design decision D1 (see the change's design.md): we shell out to the system
// `git` binary as the primary path rather than embedding go-git. This mirrors
// how the GitHub CLI works, keeps the dependency graph small, and avoids
// go-git's slow paths on large staged diffs. The trade-off — git must be on
// PATH at runtime — is an accepted, explicit requirement.
package gitio

// ChangeKind classifies how a file changed in the diff.
type ChangeKind string

const (
	KindAdded    ChangeKind = "added"
	KindModified ChangeKind = "modified"
	KindDeleted  ChangeKind = "deleted"
	KindRenamed  ChangeKind = "renamed"
)

// FileChange is one file's contribution to the diff. Binary files report
// Added/Removed as 0 (git emits "-" for them) but still count as one changed
// file, matching the diff-extraction spec.
type FileChange struct {
	Path    string
	OldPath string // set only for renames
	Kind    ChangeKind
	Added   int
	Removed int
	Binary  bool
}

// Diff is the whole change under inspection. It is the input every gate sees.
type Diff struct {
	Files []FileChange
}

// ChangedLines is the total of added + removed across all files — the quantity
// the diff-budget gate measures against max_lines.
func (d *Diff) ChangedLines() int {
	total := 0
	for _, f := range d.Files {
		total += f.Added + f.Removed
	}
	return total
}

// NewFiles counts files added by this diff (against max_new_files).
func (d *Diff) NewFiles() int {
	n := 0
	for _, f := range d.Files {
		if f.Kind == KindAdded {
			n++
		}
	}
	return n
}

// Empty reports whether there is anything to check.
func (d *Diff) Empty() bool { return len(d.Files) == 0 }
