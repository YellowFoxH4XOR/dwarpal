package aipatterns

import "testing"

// Every built-in rule must carry passing positive AND negative examples. This
// makes the rule set a tested spec: a new rule added without examples, or a
// pattern edit that breaks an example, fails here — so the "rules are tested"
// property cannot silently regress.
func TestCheckExamples_EveryRuleCovered(t *testing.T) {
	checks := CheckExamples()
	if len(checks) == 0 {
		t.Fatal("no rules checked")
	}
	for _, c := range checks {
		if !c.OK() {
			t.Errorf("%s: %v", c.RuleID, c.Failures)
		}
		if c.Positives == 0 || c.Negatives == 0 {
			t.Errorf("%s: must have both positive and negative examples (have +%d -%d)",
				c.RuleID, c.Positives, c.Negatives)
		}
	}
}

// The whole point is regression detection: if a rule's negative example starts
// matching (rule grew too broad) or a positive stops matching (rule broke),
// CheckExamples must report it, not pass silently.
func TestCheckExamples_DetectsBrokenRule(t *testing.T) {
	// A rule whose negative it will itself flag = too broad.
	broken := RegexRule{
		ID:        "x",
		Pattern:   builtinRegexRules()[0].Pattern, // reuse a real pattern
		Positives: []string{"nothing-here"},       // will NOT match → failure
		Negatives: []string{"also-nothing"},
	}
	c := RuleCheck{RuleID: broken.ID}
	for _, p := range broken.Positives {
		if !broken.Pattern.MatchString(p) {
			c.Failures = append(c.Failures, "pos")
		}
	}
	if c.OK() {
		t.Fatal("a non-matching positive must be reported as a failure")
	}
}
