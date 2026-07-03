// Package aipatterns implements Gate 3 — the AI-pattern rule pack.
//
// It targets documented agent failure modes (PRD §1): rule-silencing
// suppressions, hardcoded secrets, and (after the tree-sitter spike) SQL
// concatenation and broad exception swallowing. Rules are data, not code, so
// the community can contribute rules without touching the engine.
//
// This package ships the regex tier (any language, no AST). The AST tier plugs
// in via the same Gate once `spike-tree-sitter-ast` lands.
package aipatterns

import (
	"context"

	"github.com/YellowFoxH4XOR/dwarpal/internal/engine"
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
// (config's disable_rules).
func New(disable []string) *Gate {
	disabled := make(map[string]bool, len(disable))
	for _, id := range disable {
		disabled[id] = true
	}
	return &Gate{regexRules: builtinRegexRules(), disabled: disabled}
}

// ID identifies the gate.
func (g *Gate) ID() string { return gateID }

// RuleIDs returns the built-in regex rule IDs, for `dwarpal rules`.
func RuleIDs() []string {
	rules := builtinRegexRules()
	ids := make([]string, len(rules))
	for i, r := range rules {
		ids[i] = r.ID
	}
	return ids
}

// Run matches every enabled regex rule against every added line, emitting a
// finding at the precise file:line for each hit. Only added lines are checked —
// pre-existing suppressions/secrets are not the agent's doing and not this
// commit's concern.
func (g *Gate) Run(_ context.Context, d *gitio.Diff, _ engine.RepoIndex) ([]finding.Finding, error) {
	var findings []finding.Finding
	for _, f := range d.Files {
		// Entropy tier of no-hardcoded-secrets: statistical detection of
		// random-looking tokens that no fixed shape rule can enumerate (#23).
		if !g.disabled["no-hardcoded-secrets/entropy"] {
			findings = append(findings, EntropyFindings(f)...)
		}
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
