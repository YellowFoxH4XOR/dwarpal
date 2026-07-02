// Package finding defines the shared result types every gate emits.
//
// Finding is the single currency of the gate pipeline: gates produce them,
// the engine aggregates them, and the report layer renders them. Keeping this
// type in its own leaf package (no dependencies on gates or the engine) lets
// every other package import it without creating cycles.
package finding

// Severity ranks a finding. Only Error blocks a commit in enforce mode; Warn
// and Info are advisory. The zero value is intentionally invalid so a finding
// constructed without a severity fails loudly rather than silently passing.
type Severity string

const (
	SeverityError Severity = "error"
	SeverityWarn  Severity = "warn"
	SeverityInfo  Severity = "info"
)

// Blocking reports whether a finding of this severity should block in enforce
// mode. Centralizing the rule here keeps the engine and report layer in sync.
func (s Severity) Blocking() bool { return s == SeverityError }

// Finding is one violation reported by a gate. The field set matches the
// output contract in PRD §5.2: {gate, rule_id, severity, file, line, message,
// suggestion, docs_url}. RetryHint is the imperative, agent-consumable
// instruction that turns a block into part of the agent's retry loop (P4).
type Finding struct {
	Gate       string   `json:"gate"`
	RuleID     string   `json:"rule_id"`
	Severity   Severity `json:"severity"`
	File       string   `json:"file,omitempty"`
	Line       int      `json:"line,omitempty"`
	Message    string   `json:"message"`
	Suggestion string   `json:"suggestion,omitempty"`
	DocsURL    string   `json:"docs_url,omitempty"`
	RetryHint  string   `json:"-"`
}
