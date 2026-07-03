package agentsetup

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUpsert_CreatesAndIsIdempotent(t *testing.T) {
	root := t.TempDir()
	path, created, err := UpsertInstructions(root, ToolCodex)
	if err != nil || !created {
		t.Fatalf("first run: created=%v err=%v", created, err)
	}
	if filepath.Base(path) != "AGENTS.md" {
		t.Fatalf("codex should write AGENTS.md, got %s", path)
	}
	first, _ := os.ReadFile(path)

	// Second run: replaced in place, byte-identical result, no duplication.
	if _, created, err = UpsertInstructions(root, ToolCodex); err != nil || created {
		t.Fatalf("second run: created=%v err=%v", created, err)
	}
	second, _ := os.ReadFile(path)
	if string(first) != string(second) {
		t.Fatal("idempotent re-run must produce identical content")
	}
	if strings.Count(string(second), beginMarker) != 1 {
		t.Fatal("managed block duplicated")
	}
}

// User content outside the markers must survive an update untouched.
func TestUpsert_PreservesSurroundingContent(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "CLAUDE.md")
	pre := "# My project\n\nHuman-written rules here.\n"
	post := "\n## After the block\nmore human content\n"
	os.WriteFile(path, []byte(pre+instructionBlock(ToolClaudeCode)+post), 0o644)

	if _, _, err := UpsertInstructions(root, ToolClaudeCode); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(path)
	if !strings.HasPrefix(string(got), pre) || !strings.HasSuffix(string(got), post) {
		t.Fatalf("surrounding content damaged:\n%s", got)
	}
}

func TestUpsert_ToolFilesAndIdentities(t *testing.T) {
	for tool, wantFile := range map[Tool]string{
		ToolClaudeCode: "CLAUDE.md",
		ToolCodex:      "AGENTS.md",
		ToolOpenCode:   "AGENTS.md",
		ToolPi:         "AGENTS.md",
	} {
		root := t.TempDir()
		path, _, err := UpsertInstructions(root, tool)
		if err != nil {
			t.Fatal(err)
		}
		if filepath.Base(path) != wantFile {
			t.Errorf("%s: file = %s, want %s", tool, filepath.Base(path), wantFile)
		}
		content, _ := os.ReadFile(path)
		if !strings.Contains(string(content), `AGENTGATE_AGENT="`+agentIdentity(tool)+`"`) {
			t.Errorf("%s: identity line missing", tool)
		}
	}
}

func TestMergeClaudeSettings_CreatesPreservesDedupes(t *testing.T) {
	root := t.TempDir()

	// Pre-existing settings with unrelated keys and an existing hook.
	dir := filepath.Join(root, ".claude")
	os.MkdirAll(dir, 0o755)
	existing := `{
  "model": "opus",
  "permissions": {"allow": ["Bash(go *)"]},
  "hooks": {"PreToolUse": [{"matcher": "Write", "hooks": [{"type": "command", "command": "echo hi"}]}]}
}`
	os.WriteFile(filepath.Join(dir, "settings.json"), []byte(existing), 0o644)

	path, added, err := MergeClaudeSettings(root)
	if err != nil || !added {
		t.Fatalf("merge: added=%v err=%v", added, err)
	}
	raw, _ := os.ReadFile(path)
	var s map[string]any
	if err := json.Unmarshal(raw, &s); err != nil {
		t.Fatal(err)
	}
	if s["model"] != "opus" {
		t.Fatal("unrelated key dropped")
	}
	pre := s["hooks"].(map[string]any)["PreToolUse"].([]any)
	if len(pre) != 2 {
		t.Fatalf("existing hook lost or dwarpal hook missing: %d entries", len(pre))
	}

	// Idempotent: second merge is a no-op.
	if _, added, _ := MergeClaudeSettings(root); added {
		t.Fatal("second merge must not add a duplicate hook")
	}

	// Invalid JSON fails loudly rather than clobbering.
	os.WriteFile(filepath.Join(dir, "settings.json"), []byte("{broken"), 0o644)
	if _, _, err := MergeClaudeSettings(root); err == nil {
		t.Fatal("invalid settings.json must error, not be overwritten")
	}
}
