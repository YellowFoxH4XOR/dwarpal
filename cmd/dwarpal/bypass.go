package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/YellowFoxH4XOR/dwarpal/internal/config"
)

// bypassLogDir is where the auditable bypass log lives, relative to the repo
// root.
const bypassLogDir = ".dwarpal"

// bypassLogFile is the log filename within bypassLogDir.
const bypassLogFile = "bypass.log"

// bypassNoteRef is the git notes ref bypass records are attached to.
const bypassNoteRef = "refs/notes/dwarpal-bypass"

// bypassRecord is one JSON line appended to the bypass log (PRD §5.1).
type bypassRecord struct {
	Timestamp string `json:"timestamp"`
	Reason    string `json:"reason"`
	Branch    string `json:"branch"`
	TreeHash  string `json:"tree_hash"`
}

func newBypassCmd() *cobra.Command {
	var reason string
	cmd := &cobra.Command{
		Use:   "bypass",
		Short: "Record a one-shot, auditable bypass of the gates",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runBypass(reason)
		},
	}
	cmd.Flags().StringVar(&reason, "reason", "", "reason for bypassing the gates (required)")
	return cmd
}

// runBypass writes an auditable bypass record: a JSON line in
// .dwarpal/bypass.log and, best-effort, a git note on HEAD. It is rejected
// outright in ci_strict mode, where local bypass has no authority (PRD §5.1).
func runBypass(reason string) error {
	if reason == "" {
		return &exitError{code: 2, msg: "--reason is required"}
	}

	root, err := repoRoot()
	if err != nil {
		return &exitError{code: 2, msg: "a git repository is required"}
	}

	cfg, err := config.Load(root)
	if err != nil {
		return &exitError{code: 2, msg: err.Error()}
	}
	if cfg.Mode == config.ModeCIStrict {
		return &exitError{code: 2, msg: "bypasses are rejected under ci_strict mode"}
	}

	treeHash, err := writeTree(root)
	if err != nil {
		return &exitError{code: 2, msg: err.Error()}
	}

	rec := bypassRecord{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Reason:    reason,
		Branch:    currentBranch(root),
		TreeHash:  treeHash,
	}
	if err := appendBypassLog(root, rec); err != nil {
		return &exitError{code: 2, msg: err.Error()}
	}

	// Best-effort: attach a git note on HEAD if HEAD exists (fresh repos with
	// no commits have no HEAD to note).
	if hasHead(root) {
		_ = addBypassNote(root, rec)
	}

	fmt.Printf("• bypass recorded: %s\n", reason)
	return nil
}

// writeTree runs `git write-tree` to capture the staged tree hash.
func writeTree(root string) (string, error) {
	cmd := exec.Command("git", "write-tree")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git write-tree: %w", err)
	}
	return trimNewline(string(out)), nil
}

// hasHead reports whether HEAD resolves to a commit (false in a fresh repo
// with no commits yet).
func hasHead(root string) bool {
	cmd := exec.Command("git", "rev-parse", "--verify", "-q", "HEAD")
	cmd.Dir = root
	return cmd.Run() == nil
}

// addBypassNote attaches rec as a git note on HEAD under bypassNoteRef.
func addBypassNote(root string, rec bypassRecord) error {
	body, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	cmd := exec.Command("git", "notes", "--ref="+bypassNoteRef, "append", "-m", string(body), "HEAD")
	cmd.Dir = root
	return cmd.Run()
}

// appendBypassLog appends rec as a JSON line to root/.dwarpal/bypass.log,
// creating the directory and file as needed.
func appendBypassLog(root string, rec bypassRecord) error {
	dir := filepath.Join(root, bypassLogDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating %s: %w", bypassLogDir, err)
	}

	line, err := json.Marshal(rec)
	if err != nil {
		return err
	}

	path := filepath.Join(dir, bypassLogFile)
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("opening %s: %w", path, err)
	}
	defer f.Close()

	if _, err := f.Write(append(line, '\n')); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}

// trimNewline strips a single trailing newline, as produced by git plumbing
// commands' Output().
func trimNewline(s string) string {
	if len(s) > 0 && s[len(s)-1] == '\n' {
		return s[:len(s)-1]
	}
	return s
}
