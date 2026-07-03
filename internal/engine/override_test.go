package engine

import (
	"context"
	"testing"

	"github.com/YellowFoxH4XOR/dwarpal/internal/finding"
	"github.com/YellowFoxH4XOR/dwarpal/internal/gitio"
)

// A severity override must change the blocking decision: an error-severity
// finding demoted to warn must stop blocking. This is the whole mechanism
// behind rule_overrides and `audit --apply`; if it doesn't flip Blocking(), the
// feature does nothing.
func TestRunWith_SeverityOverrideDemotesAndUnblocks(t *testing.T) {
	ran := false
	gate := stubGate{"g1", block("g1"), &ran} // emits gate=g1 rule=r severity=error

	// Without an override, it blocks.
	res := RunWith(context.Background(), []Gate{gate}, &gitio.Diff{}, NoIndex{}, Options{})
	if !res.Blocking() {
		t.Fatal("error finding should block without an override")
	}

	// Demote g1/r to warn — it must no longer block.
	res = RunWith(context.Background(), []Gate{gate}, &gitio.Diff{}, NoIndex{},
		Options{SeverityOverrides: map[string]finding.Severity{"g1/r": finding.SeverityWarn}})
	if res.Blocking() {
		t.Fatal("override to warn must stop the finding from blocking")
	}
	if len(res.Findings) != 1 || res.Findings[0].Severity != finding.SeverityWarn {
		t.Fatalf("finding should still be reported, at warn: %+v", res.Findings)
	}
}

// The demotion must also work in StopOnFirstBlock mode — where the block check
// happens DURING the run — or a demoted rule would still short-circuit the
// pipeline. This is exactly why overrides are applied inside the engine.
func TestRunWith_OverrideHonoredUnderStopOnFirstBlock(t *testing.T) {
	first, second := false, false
	res := RunWith(context.Background(), []Gate{
		stubGate{"g1", block("g1"), &first},
		stubGate{"g2", block("g2"), &second},
	}, &gitio.Diff{}, NoIndex{}, Options{
		StopOnFirstBlock:  true,
		SeverityOverrides: map[string]finding.Severity{"g1/r": finding.SeverityWarn},
	})
	// g1 demoted to warn → not blocking → g2 must still run and block.
	if !second {
		t.Fatal("second gate should run after the first is demoted below blocking")
	}
	if !res.Blocking() {
		t.Fatal("g2's error should still block")
	}
}
