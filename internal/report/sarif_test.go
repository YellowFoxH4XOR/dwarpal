package report

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/YellowFoxH4XOR/dwarpal/internal/finding"
)

func TestSARIF_ShapeAndLevels(t *testing.T) {
	in := Input{
		Result: ResultBlocked,
		Findings: []finding.Finding{
			{Gate: "diff_budget", RuleID: "max-lines", Severity: finding.SeverityError, Message: "too big"},
			{Gate: "ai_patterns", RuleID: "no-broad-catch", Severity: finding.SeverityWarn, Message: "bare catch", File: "a.go", Line: 12},
		},
	}
	var buf bytes.Buffer
	if err := SARIF(&buf, in); err != nil {
		t.Fatal(err)
	}

	// Must be valid JSON with the SARIF 2.1.0 skeleton.
	var log map[string]any
	if err := json.Unmarshal(buf.Bytes(), &log); err != nil {
		t.Fatalf("not valid JSON: %v", err)
	}
	if log["version"] != "2.1.0" {
		t.Errorf("version = %v, want 2.1.0", log["version"])
	}

	out := buf.String()
	// Severity mapping: error->error, warn->warning.
	if !strings.Contains(out, `"level": "error"`) || !strings.Contains(out, `"level": "warning"`) {
		t.Errorf("severity levels not mapped:\n%s", out)
	}
	// Rule IDs are namespaced by gate.
	if !strings.Contains(out, `"ruleId": "diff_budget/max-lines"`) {
		t.Errorf("ruleId not namespaced by gate:\n%s", out)
	}
	// File:line becomes a physical location with startLine.
	if !strings.Contains(out, `"startLine": 12`) || !strings.Contains(out, `"uri": "a.go"`) {
		t.Errorf("location not emitted:\n%s", out)
	}
}

// Regression: EVERY result must carry at least one location, or GitHub Code
// Scanning rejects the whole file ("locationFromSarifResult: expected at least
// one location"). File-less findings (diff_budget, branch_policy describe the
// whole change, not one file) are anchored to the policy file. Caught live by
// the dogfood Action gate on the strip-to-wedge PR, whose oversized diff was the
// first to trip a file-less diff_budget finding under --sarif.
func TestSARIF_EveryResultHasLocation(t *testing.T) {
	in := Input{
		Result: ResultBlocked,
		Findings: []finding.Finding{
			{Gate: "diff_budget", RuleID: "max-lines", Severity: finding.SeverityError, Message: "too big"},
			{Gate: "branch_policy", RuleID: "protected-branch", Severity: finding.SeverityError, Message: "agent on main"},
			{Gate: "ai_patterns", RuleID: "no-broad-catch", Severity: finding.SeverityWarn, Message: "bare catch", File: "a.go", Line: 12},
		},
	}
	var buf bytes.Buffer
	if err := SARIF(&buf, in); err != nil {
		t.Fatal(err)
	}

	var log struct {
		Runs []struct {
			Results []struct {
				RuleID    string `json:"ruleId"`
				Locations []any  `json:"locations"`
			} `json:"results"`
		} `json:"runs"`
	}
	if err := json.Unmarshal(buf.Bytes(), &log); err != nil {
		t.Fatalf("not valid JSON: %v", err)
	}
	for _, r := range log.Runs[0].Results {
		if len(r.Locations) == 0 {
			t.Fatalf("result %q has no location — CodeQL rejects this", r.RuleID)
		}
	}
	// File-less findings anchor to the policy file.
	if !strings.Contains(buf.String(), `"uri": "`+diffLevelAnchor+`"`) {
		t.Fatalf("file-less finding not anchored to %s:\n%s", diffLevelAnchor, buf.String())
	}
}

// Regression: a zero-finding run must marshal rules/results as [] (not null) —
// GitHub's SARIF upload rejects "rules is not of a type(s) array". Caught live
// by the first real Action run on PR #3.
func TestSARIF_EmptyRunIsValidArrays(t *testing.T) {
	var buf bytes.Buffer
	if err := SARIF(&buf, Input{Result: ResultPassed}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, `"rules": []`) {
		t.Fatalf("empty run must emit rules: [], got:\n%s", out)
	}
	if !strings.Contains(out, `"results": []`) {
		t.Fatalf("empty run must emit results: [], got:\n%s", out)
	}
}
