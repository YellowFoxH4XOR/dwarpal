package agentsetup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The skill must land where each agent actually scans for it, or it's invisible
// — Claude Code reads .claude/skills; Codex/OpenCode/Pi share .agents/skills.
// A wrong path is a silent no-op, so pin the mapping.
func TestUpsertSkill_PathsPerTool(t *testing.T) {
	for tool, wantDir := range map[Tool]string{
		ToolClaudeCode: ".claude/skills/dwarpal",
		ToolCodex:      ".agents/skills/dwarpal",
		ToolOpenCode:   ".agents/skills/dwarpal",
		ToolPi:         ".agents/skills/dwarpal",
	} {
		root := t.TempDir()
		path, created, err := UpsertSkill(root, tool)
		if err != nil || !created {
			t.Fatalf("%s: created=%v err=%v", tool, created, err)
		}
		want := filepath.Join(root, wantDir, "SKILL.md")
		if path != want {
			t.Errorf("%s: path = %s, want %s", tool, path, want)
		}
		if _, err := os.Stat(path); err != nil {
			t.Errorf("%s: file not written: %v", tool, err)
		}
	}
}

// Codex, OpenCode, and Pi deliberately share one .agents/skills file (all three
// scan that path), so setting up a second of them must land on the SAME file,
// not scatter duplicates — mirroring how the AGENTS.md block is shared.
func TestUpsertSkill_AgentsToolsShareOneFile(t *testing.T) {
	root := t.TempDir()
	p1, created1, _ := UpsertSkill(root, ToolCodex)
	if !created1 {
		t.Fatal("codex: expected a fresh install")
	}
	p2, created2, _ := UpsertSkill(root, ToolOpenCode)
	if p1 != p2 {
		t.Fatalf("codex and opencode should share one skill file: %s vs %s", p1, p2)
	}
	if created2 {
		t.Error("opencode should refresh the shared file, not report a new create")
	}
}

// The frontmatter must carry the universal name+description every agent's
// parser needs; a missing description means no auto-invocation (silent
// degradation). The body must teach the two load-bearing behaviors — pre-flight
// check and config authoring — since that's the skill's reason to exist.
func TestSkillDoc_FrontmatterAndIntent(t *testing.T) {
	doc := skillDoc(ToolCodex)
	if !strings.HasPrefix(doc, "---\nname: "+skillName+"\n") {
		t.Error("skill must open with YAML frontmatter naming the skill")
	}
	for _, must := range []string{
		"description:",
		"dwarpal check --explain-for-agent", // the pre-flight loop
		"config author",                     // config authoring
		"Never bypass",                      // the non-negotiable
	} {
		if !strings.Contains(doc, must) {
			t.Errorf("skill body missing %q — its core purpose", must)
		}
	}
}

// Re-running setup must refresh content in place, not report a spurious new
// install — the file is the whole identity, so a second run is an update.
func TestUpsertSkill_Idempotent(t *testing.T) {
	root := t.TempDir()
	if _, created, _ := UpsertSkill(root, ToolClaudeCode); !created {
		t.Fatal("first run should create")
	}
	path, created, err := UpsertSkill(root, ToolClaudeCode)
	if err != nil {
		t.Fatal(err)
	}
	if created {
		t.Error("second run must report a refresh, not a create")
	}
	got, _ := os.ReadFile(path)
	if string(got) != skillDoc(ToolClaudeCode) {
		t.Error("re-run must leave the canonical skill content")
	}
}
