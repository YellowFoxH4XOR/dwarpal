package plugin

import "testing"

func TestParseToolJSON_GitleaksStyleArray(t *testing.T) {
	out := []byte(`[
		{"file": "config/secrets.go", "line": 12, "rule_id": "generic-api-key", "description": "hardcoded API key"},
		{"File": "src/main.go", "StartLine": 5, "Description": "AWS secret found"}
	]`)

	fs, ok := ParseToolJSON("gitleaks", out)
	if !ok {
		t.Fatalf("expected ok=true for gitleaks-style array")
	}
	if len(fs) != 2 {
		t.Fatalf("expected 2 findings, got %d: %+v", len(fs), fs)
	}

	f0 := fs[0]
	if f0.Gate != "plugin/gitleaks" || f0.File != "config/secrets.go" || f0.Line != 12 {
		t.Errorf("finding 0 mismatched: %+v", f0)
	}
	if f0.RuleID != "generic-api-key" {
		t.Errorf("expected rule_id to carry through, got %q", f0.RuleID)
	}
	if f0.Message != "generic-api-key: hardcoded API key" {
		t.Errorf("expected rule_id+description message, got %q", f0.Message)
	}
	if f0.Severity != "error" {
		t.Errorf("expected severity error, got %q", f0.Severity)
	}

	f1 := fs[1]
	if f1.File != "src/main.go" || f1.Line != 5 {
		t.Errorf("finding 1 mismatched: %+v", f1)
	}
	if f1.Message != "AWS secret found" {
		t.Errorf("expected Description as message, got %q", f1.Message)
	}
	if f1.RuleID != "finding" {
		t.Errorf("expected fallback rule_id 'finding', got %q", f1.RuleID)
	}
}

func TestParseToolJSON_SemgrepStyleResults(t *testing.T) {
	out := []byte(`{
		"results": [
			{
				"check_id": "python.lang.security.audit.eval-detected",
				"path": "app/utils.py",
				"start": {"line": 42},
				"location": {"path": "app/utils.py", "start": {"line": 42}}
			}
		],
		"errors": []
	}`)

	fs, ok := ParseToolJSON("semgrep", out)
	if !ok {
		t.Fatalf("expected ok=true for semgrep-style results object")
	}
	if len(fs) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(fs), fs)
	}
	f := fs[0]
	if f.Gate != "plugin/semgrep" {
		t.Errorf("expected gate plugin/semgrep, got %q", f.Gate)
	}
	if f.File != "app/utils.py" {
		t.Errorf("expected file from location.path, got %q", f.File)
	}
	if f.Line != 42 {
		t.Errorf("expected line from location.start.line, got %d", f.Line)
	}
	if f.Message != "python.lang.security.audit.eval-detected" {
		t.Errorf("expected check_id as message, got %q", f.Message)
	}
	if f.RuleID != "python.lang.security.audit.eval-detected" {
		t.Errorf("expected check_id as rule_id, got %q", f.RuleID)
	}
}

func TestParseToolJSON_FindingsKeyVariant(t *testing.T) {
	out := []byte(`{"findings": [{"file": "a.go", "line": 1, "message": "issue"}]}`)
	fs, ok := ParseToolJSON("tool", out)
	if !ok || len(fs) != 1 {
		t.Fatalf("expected one finding from 'findings' key, got ok=%v fs=%+v", ok, fs)
	}
}

func TestParseToolJSON_GarbageInput(t *testing.T) {
	cases := [][]byte{
		[]byte("not json at all"),
		[]byte(`{"unrelated": "field"}`),
		[]byte(`[]`),
		[]byte(`[{"no_file_field": true}]`),
		[]byte(""),
	}
	for _, c := range cases {
		if fs, ok := ParseToolJSON("tool", c); ok || fs != nil {
			t.Errorf("expected (nil, false) for garbage input %q, got (%+v, %v)", c, fs, ok)
		}
	}
}
