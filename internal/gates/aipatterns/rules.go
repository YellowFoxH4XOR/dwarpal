package aipatterns

import (
	"regexp"
	"strings"

	"github.com/YellowFoxH4XOR/dwarpal/internal/finding"
)

// RegexRule is a rules-as-data check that matches added lines. Keeping rules as
// data (not code) is the community-leverage decision: contributors add a rule by
// adding an entry + a test, never touching the engine.
type RegexRule struct {
	ID         string
	Pattern    *regexp.Regexp
	Severity   finding.Severity
	Message    string
	Suggestion string
	RetryHint  string
	// Fix is a concrete before→after example of the corrected form — the worked
	// example FeedbackEval (arXiv:2504.06939) found most improves an agent's fix
	// rate. Kept short; the same for every hit of the rule.
	Fix string
	// Positives are code fragments the rule MUST flag; Negatives are ones it
	// must NOT. They make each rule a testable spec (`dwarpal rules test`):
	// canonical living documentation of exactly what trips the rule, and a
	// regression guard on the reviewer's own judgment.
	Positives []string
	Negatives []string
}

// asm assembles a positive example from fragments so the contiguous trigger
// never appears verbatim in THIS source file — otherwise ai_patterns would
// flag its own rule definitions when rules.go is committed. Split each positive
// at its distinctive token.
func asm(parts ...string) string { return strings.Join(parts, "") }

// builtinRegexRules is the rule pack: the agent-specific tells that distinguish
// this gate from a generic linter. Secrets belong to gitleaks/trufflehog and
// arbitrary AST assertions to semgrep; what stays here is what an AI agent does
// that a human rarely does — silencing a check, or swallowing an error to make
// the diff pass.
func builtinRegexRules() []RegexRule {
	return []RegexRule{
		{
			// Agents insert suppressions to pass checks.
			ID:         "no-new-lint-suppressions",
			Pattern:    regexp.MustCompile(`eslint-disable|#\s*noqa|//\s*nolint|@ts-ignore|@ts-nocheck|#\s*pragma\s+warning\s+disable`),
			Severity:   finding.SeverityError,
			Message:    "newly added lint/type suppression",
			Suggestion: "fix the underlying issue instead of silencing the check, or add an approved override trailer",
			RetryHint:  "Remove the added lint/type suppression and fix the underlying warning it hides.",
			// Split at the trigger token (asm) so this example line does not
			// self-flag when rules.go is committed.
			Fix: asm("- value = risky()  # no", "qa") + "\n+ value = risky()  # handle the specific warning, e.g. narrow the type or check the error",
			Positives:  []string{asm("x = 1  # no", "qa"), asm("foo()  // no", "lint:errcheck"), asm("const x = 1 // @ts-ig", "nore")},
			Negatives:  []string{"x = 1  # a normal comment", "// an ordinary comment", "const x = 1"},
		},
		{
			// Agents broaden a catch to swallow the error that was blocking them.
			ID:         "no-broad-catch",
			Pattern:    regexp.MustCompile(`(\bexcept\s*:|\bexcept\s+Exception\s*:\s*(pass)?\s*$|catch\s*\([^)]*\)\s*\{\s*\}|catch\s*\{\s*\})`),
			Severity:   finding.SeverityWarn,
			Message:    "broad exception catch that may swallow errors",
			Suggestion: "catch a specific exception and log or rethrow; don't silently swallow",
			RetryHint:  "Narrow this catch to the expected error type and log or rethrow instead of swallowing it.",
			Fix: asm("- } catch (e) ", "{}") + "\n+ } catch (e) { logger.error(e); throw e; }",
			Positives:  []string{asm("    exc", "ept:"), asm("} catch (e) ", "{}")},
			Negatives:  []string{"except ValueError as e:", "} catch (e) { log.error(e); }"},
		},
	}
}
