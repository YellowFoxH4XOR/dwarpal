package aipatterns

// Example-tested rules: each built-in regex rule ships positive examples it MUST
// flag and negative examples it must NOT. Verifying them turns the rule set into
// a tested spec — a regression guard on the reviewer's own judgment and an
// explicit defense of the false-positive budget (a negative that wrongly matches
// is a rule that is too broad, caught before it ever annoys a developer).

// RuleCheck is the result of testing one rule against its examples.
type RuleCheck struct {
	RuleID    string   `json:"rule_id"`
	Severity  string   `json:"severity"`
	Positives int      `json:"positives"`
	Negatives int      `json:"negatives"`
	Failures  []string `json:"failures,omitempty"`
}

// OK reports whether the rule is fully covered and every example behaved.
func (c RuleCheck) OK() bool { return len(c.Failures) == 0 }

// CheckExamples verifies every built-in regex rule against its examples: each
// positive MUST match, each negative MUST NOT, and a rule lacking either kind
// of example is a coverage gap (reported as a failure so the spec stays whole).
func CheckExamples() []RuleCheck {
	rules := builtinRegexRules()
	out := make([]RuleCheck, 0, len(rules))
	for _, r := range rules {
		c := RuleCheck{RuleID: r.ID, Severity: string(r.Severity), Positives: len(r.Positives), Negatives: len(r.Negatives)}
		if len(r.Positives) == 0 {
			c.Failures = append(c.Failures, "no positive examples — rule is untested")
		}
		if len(r.Negatives) == 0 {
			c.Failures = append(c.Failures, "no negative examples — false-positive behavior is untested")
		}
		for _, p := range r.Positives {
			if !r.Pattern.MatchString(p) {
				c.Failures = append(c.Failures, "positive not flagged: "+p)
			}
		}
		for _, n := range r.Negatives {
			if r.Pattern.MatchString(n) {
				c.Failures = append(c.Failures, "negative wrongly flagged: "+n)
			}
		}
		out = append(out, c)
	}
	return out
}
