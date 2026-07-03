package provenance

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// setupRepo creates a temp git repo with a single commit and returns its
// path. It pins git's global/system config to /dev/null and injects a
// deterministic author/committer identity via t.Setenv, so the exec.Command
// calls inside AttachNote (which inherit the test process's environment)
// behave hermetically regardless of the host machine's git config — the same
// pattern used in internal/gitio/extract_test.go.
func setupRepo(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_SYSTEM", os.DevNull)
	t.Setenv("GIT_AUTHOR_NAME", "t")
	t.Setenv("GIT_AUTHOR_EMAIL", "t@t.co")
	t.Setenv("GIT_COMMITTER_NAME", "t")
	t.Setenv("GIT_COMMITTER_EMAIL", "t@t.co")

	dir := t.TempDir()
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	git("init")
	if err := os.WriteFile(filepath.Join(dir, "f.txt"), []byte("hi\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git("add", "f.txt")
	git("commit", "-m", "base")
	return dir
}

// TestAttachNote_AttachesJSONNote verifies the happy path end to end against
// a real repo: the note must exist on the provenance ref and contain the
// agent name and source, since that JSON is the contract downstream audit
// tooling reads.
func TestAttachNote_AttachesJSONNote(t *testing.T) {
	dir := setupRepo(t)

	if err := AttachNote(dir, Provenance{IsAgent: true, Source: SourceBranch, Agent: "claude-code"}); err != nil {
		t.Fatalf("AttachNote: %v", err)
	}

	cmd := exec.Command("git", "notes", "--ref=dwarpal-provenance", "show", "HEAD")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git notes show: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "claude-code") {
		t.Errorf("note missing agent name: %s", out)
	}
	if !strings.Contains(string(out), `"source":"branch"`) {
		t.Errorf("note missing source: %s", out)
	}
}

// TestAttachNote_NotAgent_NoOp verifies human commits are left untouched —
// the core guarantee that lets Dwarpal apply gates to agent commits only
// without hook fatigue on every human commit.
func TestAttachNote_NotAgent_NoOp(t *testing.T) {
	dir := setupRepo(t)

	if err := AttachNote(dir, Provenance{IsAgent: false}); err != nil {
		t.Fatalf("AttachNote: %v", err)
	}

	cmd := exec.Command("git", "notes", "--ref=dwarpal-provenance", "list")
	cmd.Dir = dir
	out, _ := cmd.CombinedOutput()
	if strings.TrimSpace(string(out)) != "" {
		t.Errorf("expected no notes for a non-agent commit, got: %s", out)
	}
}

// TestAttachNote_FreshRepo_NoHEAD_NoOp verifies a repo with no commits yet
// (HEAD unresolved) degrades to a silent no-op instead of erroring, since
// there is no commit to attach a note to.
func TestAttachNote_FreshRepo_NoHEAD_NoOp(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_SYSTEM", os.DevNull)

	dir := t.TempDir()
	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}

	if err := AttachNote(dir, Provenance{IsAgent: true, Source: SourceEnv, Agent: "x"}); err != nil {
		t.Fatalf("AttachNote on a fresh repo should no-op, got err: %v", err)
	}
}
