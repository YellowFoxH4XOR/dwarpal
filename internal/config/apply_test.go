package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// PatchRuleOverrides must add the override AND leave the user's comments and
// other settings intact — it tunes a config, it doesn't rewrite it. If comments
// are lost, users won't trust `audit --apply` to touch their file.
func TestPatchRuleOverrides_PreservesCommentsAndMerges(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, Filename)
	original := `version: 1
mode: enforce            # enforce | warn | ci_strict

gates:
  diff_budget:
    max_lines: 500       # keep commits reviewable
`
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := PatchRuleOverrides(dir, map[string]string{"ai_patterns/no-sql-concat": "warn"}); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(path)
	s := string(got)

	if !strings.Contains(s, "# enforce | warn | ci_strict") || !strings.Contains(s, "# keep commits reviewable") {
		t.Errorf("comments were lost:\n%s", s)
	}
	if !strings.Contains(s, "rule_overrides:") || !strings.Contains(s, "ai_patterns/no-sql-concat") {
		t.Errorf("override not written:\n%s", s)
	}

	// The patched file must still load and carry the override.
	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("patched config no longer loads: %v", err)
	}
	if cfg.RuleOverrides["ai_patterns/no-sql-concat"] != "warn" {
		t.Errorf("override didn't round-trip through Load: %v", cfg.RuleOverrides)
	}
}

// A second apply of the same key must update in place, not duplicate it (which
// would be invalid YAML) — re-running audit --apply has to be idempotent.
func TestPatchRuleOverrides_Idempotent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, Filename)
	os.WriteFile(path, []byte("version: 1\n"), 0o644)

	for i := 0; i < 2; i++ {
		if err := PatchRuleOverrides(dir, map[string]string{"ai_patterns/x": "warn"}); err != nil {
			t.Fatal(err)
		}
	}
	got, _ := os.ReadFile(path)
	if n := strings.Count(string(got), "ai_patterns/x"); n != 1 {
		t.Errorf("key written %d times, want 1:\n%s", n, got)
	}
	if _, err := Load(dir); err != nil {
		t.Fatalf("doubly-patched config must still be valid YAML: %v", err)
	}
}

// audit --apply tunes an existing config; with no file it must fail clearly
// rather than silently create one.
func TestPatchRuleOverrides_NoFile(t *testing.T) {
	if err := PatchRuleOverrides(t.TempDir(), map[string]string{"a/b": "warn"}); err == nil {
		t.Fatal("expected an error when .dwarpal.yml is absent")
	}
}
