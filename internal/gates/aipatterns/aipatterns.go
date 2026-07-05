// Package aipatterns implements the AI-pattern rule pack — the agent-specific
// tells (rule-silencing suppressions, error-swallowing catches) that a generic
// linter doesn't look for. Rules are data, not code, so contributors add a rule
// by adding an entry plus its positive/negative examples, never touching the
// engine. The pack is regex-only: any language, no AST, no repo index.
package aipatterns

import (
	"context"

	"github.com/YellowFoxH4XOR/dwarpal/internal/finding"
	"github.com/YellowFoxH4XOR/dwarpal/internal/gitio"
)

const gateID = "ai_patterns"

// Gate runs the enabled rule pack against added lines.
type Gate struct {
	regexRules []RegexRule
	disabled   map[string]bool
}

// New builds the gate with the built-in rules, minus any disabled by ID
// (config's disable_rules plus per-run overrides).
func New(disable []string) *Gate {
	disabled := make(map[string]bool, len(disable))
	for _, id := range disable {
		disabled[id] = true
	}
	return &Gate{regexRules: builtinRegexRules(), disabled: disabled}
}

// ID identifies the gate.
func (g *Gate) ID() string { return gateID }

// RuleIDs returns the built-in rule IDs, for `dwarpal rules`.
func RuleIDs() []string {
	rules := builtinRegexRules()
	ids := make([]string, len(rules))
	for i, r := range rules {
		ids[i] = r.ID
	}
	return ids
}

// Run matches every enabled rule against every added line, emitting a finding at
// the precise file:line for each hit. Only added lines are checked — a
// pre-existing suppression is not this commit's doing and not its concern.
func (g *Gate) Run(_ context.Context, d *gitio.Diff) ([]finding.Finding, error) {
	var findings []finding.Finding
	for _, f := range d.Files {
		for _, line := range f.AddedLines {
			for _, rule := range g.regexRules {
				if g.disabled[rule.ID] || !rule.Pattern.MatchString(line.Text) {
					continue
				}
				findings = append(findings, finding.Finding{
					Gate:       gateID,
					RuleID:     rule.ID,
					Severity:   rule.Severity,
					File:       f.Path,
					Line:       line.Number,
					Message:    rule.Message,
					Suggestion: rule.Suggestion,
					RetryHint:  rule.RetryHint,
				})
			}
		}
	}
	return findings, nil
}
