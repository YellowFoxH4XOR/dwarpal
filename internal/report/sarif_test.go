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
