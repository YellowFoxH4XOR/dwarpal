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

// Finding is one violation reported by a gate. Its agent-facing fields form the
// "mixed feedback" shape that measurably lifts an LLM's fix rate over a bare
// error message (FeedbackEval, arXiv:2504.06939 — mixed vs minimal feedback is
// +10.5pp, and dropping the worked example degrades it most): Message says WHAT
// is wrong, File/Line say WHERE, Suggestion says WHY/how, RetryHint is the
// imperative fix instruction, and Fix is a concrete before→after example.
type Finding struct {
	Gate       string   `json:"gate"`
	RuleID     string   `json:"rule_id"`
	Severity   Severity `json:"severity"`
	File       string   `json:"file,omitempty"`
	Line       int      `json:"line,omitempty"`
	Message    string   `json:"message"`
	Suggestion string   `json:"suggestion,omitempty"`
	DocsURL    string   `json:"docs_url,omitempty"`
	// RetryHint is surfaced both per-finding and in the top-level retry_hints
	// digest; Fix is the worked example FeedbackEval found most load-bearing.
	RetryHint string `json:"retry_hint,omitempty"`
	Fix       string `json:"fix,omitempty"`
}
