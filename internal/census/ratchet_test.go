package census

import (
	"testing"
)

func baseline(name string, count int, items ...string) *Baseline {
	return &Baseline{Version: 1, Detectors: map[string]BaselineEntry{
		name: {Count: count, Items: items},
	}}
}

func report(name string, count int, items ...string) *Report {
	return &Report{Detectors: []DetectorStat{{Name: name, Count: count, Items: items}}}
}

// An increase must produce one finding per genuinely-new item — and ONLY the
// new ones. Grandfathered debt (items already in the baseline) must never be
// re-reported, or every PR would drown in pre-existing findings and the ratchet
// would be ignored. This is the core promise of the design.
func TestDiff_blocksOnlyNewItems(t *testing.T) {
	base := baseline("deadcode", 1, "pkg.old")
	rep := report("deadcode", 2, "pkg.old", "pkg.new")

	findings := Diff(base, rep)
	if len(findings) != 1 {
		t.Fatalf("got %d findings, want 1 (only the new item)", len(findings))
	}
	if got := findings[0].Message; got != "new: pkg.new" {
		t.Fatalf("message = %q, want it to name only pkg.new", got)
	}
	if !findings[0].Severity.Blocking() {
		t.Fatal("ratchet finding must block")
	}
}

// A decrease is an improvement; it must never block. The number can only
// ratchet down over time.
func TestDiff_decreaseNeverBlocks(t *testing.T) {
	base := baseline("deadcode", 3, "a", "b", "c")
	rep := report("deadcode", 1, "a")
	if f := Diff(base, rep); len(f) != 0 {
		t.Fatalf("decrease produced %d findings, want 0", len(f))
	}
}

// Count is the gate (strategy §3: "fail only if the count went up"). Swapping
// one dead symbol for another at the SAME count is a wash, not new debt, so it
// must pass — otherwise routine refactors that relocate dead code would be
// blocked for no net decay.
func TestDiff_sameCountDifferentItemsPasses(t *testing.T) {
	base := baseline("deadcode", 1, "pkg.old")
	rep := report("deadcode", 1, "pkg.different")
	if f := Diff(base, rep); len(f) != 0 {
		t.Fatalf("same-count swap produced %d findings, want 0", len(f))
	}
}

// A detector with no baseline entry (first ever run) is treated as a zero
// baseline, so its current debt blocks and must be explicitly accepted via
// --update-baseline. Silently passing would let a whole category of decay in
// unmeasured.
func TestDiff_missingBaselineEntryBlocks(t *testing.T) {
	rep := report("deadcode", 2, "pkg.a", "pkg.b")
	findings := Diff(nil, rep)
	if len(findings) != 2 {
		t.Fatalf("got %d findings, want 2 (no baseline → all new)", len(findings))
	}
}

// When the count rises but item identities are unavailable (degraded parser),
// the ratchet must STILL block — reporting the delta — rather than silently
// passing because it couldn't name names.
func TestDiff_countRoseWithoutItemsStillBlocks(t *testing.T) {
	base := baseline("dupes", 5) // count only, no items
	rep := report("dupes", 8)    // count only, no items
	findings := Diff(base, rep)
	if len(findings) != 1 {
		t.Fatalf("got %d findings, want 1 delta finding", len(findings))
	}
	if findings[0].Message == "" || !findings[0].Severity.Blocking() {
		t.Fatalf("degraded finding must block and carry a message, got %+v", findings[0])
	}
}

// Skipped detectors are Run's way of recording "could not measure"; Diff must
// ignore them (the command layer fails loud on them separately). If Diff
// counted a skipped detector as zero it could mask a real increase.
func TestDiff_ignoresSkippedDetectors(t *testing.T) {
	base := baseline("deadcode", 5, "a")
	rep := &Report{Detectors: []DetectorStat{{Name: "deadcode", Skipped: true, SkipReason: "not installed"}}}
	if f := Diff(base, rep); len(f) != 0 {
		t.Fatalf("skipped detector produced %d findings, want 0", len(f))
	}
}

// WriteBaseline then LoadBaseline must round-trip exactly, and a skipped
// detector must be EXCLUDED — baking an unmeasured count into the baseline
// would let a later run "improve" against a phantom number.
func TestBaselineRoundTrip_excludesSkipped(t *testing.T) {
	root := t.TempDir()
	rep := &Report{Detectors: []DetectorStat{
		{Name: "deadcode", Count: 2, Items: []string{"pkg.b", "pkg.a"}},
		{Name: "vulture", Skipped: true, SkipReason: "not installed"},
	}}
	if err := WriteBaseline(root, DefaultBaselinePath, rep); err != nil {
		t.Fatalf("write: %v", err)
	}
	got, err := LoadBaseline(root, DefaultBaselinePath)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if _, ok := got.Detectors["vulture"]; ok {
		t.Fatal("skipped detector was baked into the baseline")
	}
	e, ok := got.Detectors["deadcode"]
	if !ok || e.Count != 2 {
		t.Fatalf("deadcode entry = %+v, want count 2", e)
	}
	// Items are sorted on write for deterministic, diff-friendly baselines.
	if e.Items[0] != "pkg.a" || e.Items[1] != "pkg.b" {
		t.Fatalf("items not sorted: %v", e.Items)
	}
}

// A missing baseline file is (nil, nil), not an error — the caller decides that
// --check without a baseline is fatal while other modes tolerate it.
func TestLoadBaseline_missingIsNilNilNotError(t *testing.T) {
	got, err := LoadBaseline(t.TempDir(), DefaultBaselinePath)
	if err != nil || got != nil {
		t.Fatalf("missing baseline: got (%v, %v), want (nil, nil)", got, err)
	}
}
