package report

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/YellowFoxH4XOR/dwarpal/internal/finding"
)

// The agent-facing JSON must carry the full "mixed feedback" set per finding —
// message (what), file/line (where), suggestion (why), retry_hint (fix
// instruction), and fix (worked example). FeedbackEval (arXiv:2504.06939) shows
// the worked example is the component whose removal degrades an LLM's fix rate
// most, so a regression that drops `fix` or `retry_hint` from the payload is a
// real loss of the differentiator — this test guards it.
func TestJSON_FindingCarriesStructuredHint(t *testing.T) {
	in := Input{
		Result: ResultBlocked,
		Findings: []finding.Finding{{
			Gate:       "ai_patterns",
			RuleID:     "no-broad-catch",
			Severity:   finding.SeverityWarn,
			File:       "a.js",
			Line:       12,
			Message:    "broad exception catch",
			Suggestion: "catch a specific exception",
			RetryHint:  "Narrow this catch and rethrow.",
			// Trigger token split so this test file does not self-flag ai_patterns.
			Fix: "- } catch (e) " + "{}\n+ } catch (e) { logger.error(e); throw e; }",
		}},
	}
	var buf bytes.Buffer
	if err := JSON(&buf, in); err != nil {
		t.Fatal(err)
	}

	var out struct {
		Findings []struct {
			RetryHint string `json:"retry_hint"`
			Fix       string `json:"fix"`
		} `json:"findings"`
		RetryHints []string `json:"retry_hints"`
	}
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("not valid JSON: %v", err)
	}
	if len(out.Findings) != 1 {
		t.Fatalf("want 1 finding, got %d", len(out.Findings))
	}
	if out.Findings[0].RetryHint == "" {
		t.Error("finding is missing retry_hint — the fix instruction the agent acts on")
	}
	if !strings.Contains(out.Findings[0].Fix, "throw e") {
		t.Errorf("finding is missing the worked fix example, got %q", out.Findings[0].Fix)
	}
	// The digest still exists for consumers that want the flat list.
	if len(out.RetryHints) != 1 {
		t.Errorf("retry_hints digest should still be populated, got %v", out.RetryHints)
	}
}

// A finding with no Fix must omit the field, not emit an empty string — keeps
// the payload clean for gates/rules that have no worked example.
func TestJSON_FixOmittedWhenAbsent(t *testing.T) {
	in := Input{
		Result:   ResultBlocked,
		Findings: []finding.Finding{{Gate: "g", RuleID: "r", Severity: finding.SeverityError, Message: "m"}},
	}
	var buf bytes.Buffer
	if err := JSON(&buf, in); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(buf.String(), `"fix"`) {
		t.Errorf("empty fix must be omitted:\n%s", buf.String())
	}
}

// TTY renders the worked example under the finding so a human sees it too.
func TestTTY_ShowsFixExample(t *testing.T) {
	in := Input{
		Result: ResultBlocked,
		Findings: []finding.Finding{{
			Gate: "ai_patterns", RuleID: "no-broad-catch", Severity: finding.SeverityWarn,
			Message: "broad catch", Fix: "- } catch (e) " + "{}\n+ } catch (e) { throw e; }",
		}},
	}
	var buf bytes.Buffer
	if err := TTY(&buf, in); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "e.g.") || !strings.Contains(buf.String(), "throw e") {
		t.Errorf("TTY should show the fix example:\n%s", buf.String())
	}
}
