// Package report renders a pipeline result for humans and machines.
//
// One findings model, N renderers (the golangci-lint printers pattern): the
// engine produces findings, and this package turns them into either a colored
// TTY report or the stable JSON contract {result, findings, summary,
// retry_hints}. SARIF will slot in here at M1 without touching the model.
package report

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"

	"github.com/YellowFoxH4XOR/dwarpal/internal/engine"
	"github.com/YellowFoxH4XOR/dwarpal/internal/finding"
)

// Result strings are part of the JSON contract.
const (
	ResultPassed  = "passed"
	ResultBlocked = "blocked"
	ResultWarned  = "warned"
)

// Input is everything the renderers need. Result is precomputed by the caller
// (it depends on mode, which the report layer deliberately does not know).
type Input struct {
	Result     string
	Findings   []finding.Finding
	GateErrors []engine.GateError
}

// summary counts findings by severity for the JSON summary block.
type summary struct {
	Findings   int `json:"findings"`
	Errors     int `json:"errors"`
	Warnings   int `json:"warnings"`
	Info       int `json:"info"`
	GateErrors int `json:"gate_errors"`
}

func (in Input) summarize() summary {
	s := summary{Findings: len(in.Findings), GateErrors: len(in.GateErrors)}
	for _, f := range in.Findings {
		switch f.Severity {
		case finding.SeverityError:
			s.Errors++
		case finding.SeverityWarn:
			s.Warnings++
		case finding.SeverityInfo:
			s.Info++
		}
	}
	return s
}

// retryHints collects the non-empty, de-duplicated agent hints in order.
func (in Input) retryHints() []string {
	seen := map[string]bool{}
	var hints []string
	for _, f := range in.Findings {
		if f.RetryHint != "" && !seen[f.RetryHint] {
			seen[f.RetryHint] = true
			hints = append(hints, f.RetryHint)
		}
	}
	return hints
}

// jsonOutput is the stable machine schema (PRD §5.4).
type jsonOutput struct {
	Result     string            `json:"result"`
	Findings   []finding.Finding `json:"findings"`
	Summary    summary           `json:"summary"`
	RetryHints []string          `json:"retry_hints"`
	GateErrors []jsonGateError   `json:"gate_errors,omitempty"`
}

type jsonGateError struct {
	Gate  string `json:"gate"`
	Error string `json:"error"`
}

// JSON writes the machine-readable result. Callers send this to stdout and keep
// all human diagnostics on stderr so stdout stays pure JSON.
func JSON(w io.Writer, in Input) error {
	findings := in.Findings
	if findings == nil {
		findings = []finding.Finding{}
	}
	hints := in.retryHints()
	if hints == nil {
		hints = []string{}
	}
	out := jsonOutput{
		Result:     in.Result,
		Findings:   findings,
		Summary:    in.summarize(),
		RetryHints: hints,
	}
	for _, ge := range in.GateErrors {
		out.GateErrors = append(out.GateErrors, jsonGateError{Gate: ge.Gate, Error: ge.Err.Error()})
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

// TTY writes the human-readable report grouped by gate.
func TTY(w io.Writer, in Input) error {
	if len(in.Findings) == 0 && len(in.GateErrors) == 0 {
		fmt.Fprintln(w, "Dwarpal: no findings — nothing blocked at the gate.")
		return nil
	}

	byGate := map[string][]finding.Finding{}
	var order []string
	for _, f := range in.Findings {
		if _, ok := byGate[f.Gate]; !ok {
			order = append(order, f.Gate)
		}
		byGate[f.Gate] = append(byGate[f.Gate], f)
	}
	sort.Strings(order)

	for _, gate := range order {
		fmt.Fprintf(w, "\n%s\n", gate)
		for _, f := range byGate[gate] {
			loc := ""
			if f.File != "" {
				loc = f.File
				if f.Line > 0 {
					loc = fmt.Sprintf("%s:%d", f.File, f.Line)
				}
				loc = " " + loc
			}
			fmt.Fprintf(w, "  [%s] %s%s\n", f.Severity, f.Message, loc)
			if f.Suggestion != "" {
				fmt.Fprintf(w, "      ↳ %s\n", f.Suggestion)
			}
		}
	}

	for _, ge := range in.GateErrors {
		fmt.Fprintf(w, "\n%s\n  [error] gate failed to run: %v\n", ge.Gate, ge.Err)
	}

	s := in.summarize()
	if in.Result == ResultBlocked {
		fmt.Fprintf(w, "\nDwarpal stopped this at the gate: %d finding(s), %d error(s).\n", s.Findings, s.Errors)
	} else {
		// Not blocked (warn mode, or only advisory findings): report, don't claim a block.
		fmt.Fprintf(w, "\nDwarpal: %d advisory finding(s) — not blocked.\n", s.Findings)
	}
	return nil
}
