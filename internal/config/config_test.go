package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeConfig drops a .dwarpal.yml into a temp dir and returns the dir.
func writeConfig(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, Filename), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestLoad_MissingFileUsesDefaults(t *testing.T) {
	cfg, err := Load(t.TempDir())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 300 is the agent-calibrated default (see Defaults) — a partial/absent
	// config must still land on it, not a stale hardcoded number.
	if cfg.Gates.DiffBudget.MaxLines != 300 || cfg.Mode != ModeEnforce {
		t.Fatalf("defaults not applied: %+v", cfg)
	}
}

// A partial file must overlay defaults, not zero the unset fields — this is the
// behavior the whole "config is an overlay" promise rests on.
func TestLoad_PartialOverlaysDefaults(t *testing.T) {
	dir := writeConfig(t, "gates:\n  diff_budget:\n    max_lines: 200\n")
	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b := cfg.Gates.DiffBudget
	if b.MaxLines != 200 {
		t.Errorf("max_lines = %d, want 200", b.MaxLines)
	}
	if b.MaxFiles != 20 || b.MaxNewFiles != 10 {
		t.Errorf("unset budgets lost their defaults: %+v", b)
	}
}

// A typo'd key must fail loudly (exit-2 path) and name the offender, so a
// misconfiguration can never silently weaken a gate.
func TestLoad_UnknownKeyRejected(t *testing.T) {
	dir := writeConfig(t, "gates:\n  diff_budget:\n    max_line: 100\n")
	_, err := Load(dir)
	if err == nil || !strings.Contains(err.Error(), "max_line") {
		t.Fatalf("want error naming max_line, got %v", err)
	}
}

func TestLoad_InvalidModeRejected(t *testing.T) {
	dir := writeConfig(t, "mode: strict\n")
	_, err := Load(dir)
	if err == nil || !strings.Contains(err.Error(), "invalid mode") {
		t.Fatalf("want invalid mode error, got %v", err)
	}
}

func TestLoad_NegativeBudgetRejected(t *testing.T) {
	dir := writeConfig(t, "gates:\n  diff_budget:\n    max_lines: -1\n")
	_, err := Load(dir)
	if err == nil || !strings.Contains(err.Error(), "negative") {
		t.Fatalf("want negative-budget error, got %v", err)
	}
}

// An unknown top-level key fails closed and names the offender, so a config for
// a removed feature (e.g. census, plugins) can't sit silently ignored.
func TestLoad_RemovedFeatureKeyRejected(t *testing.T) {
	dir := writeConfig(t, "census:\n  detectors: [deadcode]\n")
	if _, err := Load(dir); err == nil || !strings.Contains(err.Error(), "unknown config key") {
		t.Fatalf("want unknown-key rejection for removed census block, got %v", err)
	}
}
