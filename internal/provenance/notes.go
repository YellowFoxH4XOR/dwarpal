package provenance

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// NotesRef is the git notes ref Dwarpal attaches provenance records under.
// Kept short (no "refs/notes/" prefix) because `git notes --ref=` accepts
// either form and resolves an unqualified name under refs/notes/ itself.
const NotesRef = "dwarpal-provenance"

// noteRecord is the JSON payload attached to HEAD as a git note, matching the
// PRD's {agent, source} provenance record shape.
type noteRecord struct {
	Agent  string `json:"agent"`
	Source Source `json:"source"`
}

// AttachNote records agent provenance for the current HEAD commit as a git
// note (refs/notes/dwarpal-provenance), so downstream tooling can see which
// commits were agent-authored without rewriting the commit message itself.
//
// It is a no-op — returning nil — when p is not agent provenance (nothing to
// record) or when HEAD does not resolve yet (a brand-new repo with no
// commits: there is nothing to attach a note to).
func AttachNote(root string, p Provenance) error {
	if !p.IsAgent {
		return nil
	}
	if err := runGit(root, "rev-parse", "--verify", "-q", "HEAD"); err != nil {
		return nil
	}

	payload, err := json.Marshal(noteRecord{Agent: p.Agent, Source: p.Source})
	if err != nil {
		return fmt.Errorf("marshal provenance note: %w", err)
	}
	return runGit(root, "notes", "--ref="+NotesRef, "append", "-m", string(payload), "HEAD")
}

// runGit runs system git in root and wraps failures with the command and its
// combined output for diagnosability.
func runGit(root string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, out)
	}
	return nil
}
