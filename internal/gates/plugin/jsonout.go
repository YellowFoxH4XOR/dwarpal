// This file adds best-effort structured-output parsing to the exec-plugin
// gate (#44). Many security tools (gitleaks, semgrep, trivy, …) can emit JSON
// instead of a bare exit code; when they do, Dwarpal can report one Finding
// per item instead of dumping raw tool output into a single Suggestion blob.
// Parsing is deliberately loose — every tool's JSON schema differs slightly —
// so we accept a documented set of field name variants and drop anything we
// can't map to a file, treating "we understood nothing" as "not our format"
// rather than an error.
package plugin

import (
	"encoding/json"
	"strconv"

	"github.com/YellowFoxH4XOR/dwarpal/internal/finding"
)

// ParseToolJSON best-effort parses out as one of two common security-tool
// JSON shapes:
//  1. a top-level JSON array of finding objects (e.g. gitleaks), or
//  2. a JSON object with a "results" or "findings" array (e.g. semgrep).
//
// It returns (nil, false) when out is not JSON, or is JSON but yields zero
// mappable findings — callers should fall back to the raw-output behavior in
// that case rather than reporting nothing.
func ParseToolJSON(pluginName string, out []byte) ([]finding.Finding, bool) {
	var raw json.RawMessage
	if err := json.Unmarshal(out, &raw); err != nil {
		return nil, false
	}

	items := extractItems(raw)
	if items == nil {
		return nil, false
	}

	gate := "plugin/" + pluginName
	var findings []finding.Finding
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		f, ok := findingFromItem(gate, m)
		if !ok {
			continue
		}
		findings = append(findings, f)
	}
	if len(findings) == 0 {
		return nil, false
	}
	return findings, true
}

// extractItems returns the list of candidate finding objects from either a
// top-level array or an object's "results"/"findings" array. It returns nil
// if raw matches neither shape.
func extractItems(raw json.RawMessage) []any {
	var arr []any
	if err := json.Unmarshal(raw, &arr); err == nil {
		return arr
	}

	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil
	}
	for _, key := range []string{"results", "findings"} {
		if v, ok := obj[key]; ok {
			if a, ok := v.([]any); ok {
				return a
			}
		}
	}
	return nil
}

// findingFromItem maps one tool-specific JSON object to a Finding across the
// documented field-name variants (spelling/casing differs by tool — gitleaks
// capitalizes, semgrep doesn't). It returns ok=false if no file could be
// determined — a finding with no file is not useful to report.
func findingFromItem(gate string, m map[string]any) (finding.Finding, bool) {
	file := lookupString(m, "file", "path", "File")
	if file == "" {
		file = lookupNestedString(m, "location", "path")
	}
	if file == "" {
		return finding.Finding{}, false
	}

	line := lookupInt(m, "line", "StartLine", "start_line")
	if line == 0 {
		line = lookupNestedInt(m, "location", "start", "line")
	}

	ruleID := lookupString(m, "rule_id", "RuleID", "check_id")

	// Message precedence per spec: message, then Description, then a
	// rule_id+description combo (only reached when neither of the above is
	// present), then check_id, then a generic fallback.
	message := lookupString(m, "message", "Description")
	if message == "" {
		if desc := lookupString(m, "description"); desc != "" && ruleID != "" {
			message = ruleID + ": " + desc
		}
	}
	if message == "" {
		message = lookupString(m, "check_id")
	}
	if message == "" {
		message = "finding reported by " + gate
	}

	if ruleID == "" {
		ruleID = "finding"
	}

	return finding.Finding{
		Gate:     gate,
		RuleID:   ruleID,
		Severity: finding.SeverityError,
		File:     file,
		Line:     line,
		Message:  message,
	}, true
}

// lookupString returns the first present candidate key's value as a string.
func lookupString(m map[string]any, keys ...string) string {
	v, ok := lookupAny(m, keys...)
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}

// lookupInt returns the first matching key's value as an int. JSON numbers
// decode as float64 via encoding/json's default any-unmarshaling, but tools
// may also emit the line number as a string.
func lookupInt(m map[string]any, keys ...string) int {
	v, ok := lookupAny(m, keys...)
	if !ok {
		return 0
	}
	switch n := v.(type) {
	case float64:
		return int(n)
	case string:
		i, _ := strconv.Atoi(n)
		return i
	default:
		return 0
	}
}

// lookupNestedString descends into a nested object (e.g. location.path).
func lookupNestedString(m map[string]any, path ...string) string {
	v, ok := descend(m, path)
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}

// lookupNestedInt descends into a nested object (e.g. location.start.line).
func lookupNestedInt(m map[string]any, path ...string) int {
	v, ok := descend(m, path)
	if !ok {
		return 0
	}
	switch n := v.(type) {
	case float64:
		return int(n)
	case string:
		i, _ := strconv.Atoi(n)
		return i
	default:
		return 0
	}
}

// descend walks a chain of nested map keys.
func descend(m map[string]any, path []string) (any, bool) {
	var cur any = m
	for _, key := range path {
		cm, ok := cur.(map[string]any)
		if !ok {
			return nil, false
		}
		v, ok := cm[key]
		if !ok {
			return nil, false
		}
		cur = v
	}
	return cur, true
}

// lookupAny returns the value of the first candidate key present in m.
func lookupAny(m map[string]any, keys ...string) (any, bool) {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			return v, true
		}
	}
	return nil, false
}
