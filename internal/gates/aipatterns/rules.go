package aipatterns

import (
	"regexp"

	"github.com/YellowFoxH4XOR/dwarpal/internal/finding"
)

// RegexRule is a rules-as-data check that matches added lines. Keeping rules as
// data (not code) is the community-leverage decision (semgrep/gitleaks model):
// contributors add a rule by adding an entry + a test, never touching the
// engine. AST-tier rules land after the tree-sitter spike as a sibling type.
type RegexRule struct {
	ID         string
	Pattern    *regexp.Regexp
	Severity   finding.Severity
	Message    string
	Suggestion string
	RetryHint  string
}

// builtinRegexRules is the regex tier of Gate 3 — the rules that need no AST and
// therefore work on any language and ship independent of the tree-sitter spike.
func builtinRegexRules() []RegexRule {
	return []RegexRule{
		{
			// Failure mode 3: agents insert suppressions to pass checks.
			ID:         "no-new-lint-suppressions",
			Pattern:    regexp.MustCompile(`eslint-disable|#\s*noqa|//\s*nolint|@ts-ignore|@ts-nocheck|#\s*pragma\s+warning\s+disable`),
			Severity:   finding.SeverityError,
			Message:    "newly added lint/type suppression",
			Suggestion: "fix the underlying issue instead of silencing the check, or add an approved override trailer",
			RetryHint:  "Remove the added lint/type suppression and fix the underlying warning it hides.",
		},
		{
			// Failure mode 4: hardcoded private keys.
			ID:         "no-hardcoded-secrets/private-key",
			Pattern:    regexp.MustCompile(`-----BEGIN (?:RSA |EC |OPENSSH |DSA |PGP )?PRIVATE KEY-----`),
			Severity:   finding.SeverityError,
			Message:    "hardcoded private key material",
			Suggestion: "load secrets from the environment or a secret manager; never commit key material",
			RetryHint:  "Remove the committed private key and load it from a secret manager or environment variable.",
		},
		{
			// Failure mode 4: AWS access key ID shape.
			ID:         "no-hardcoded-secrets/aws-key",
			Pattern:    regexp.MustCompile(`\bAKIA[0-9A-Z]{16}\b`),
			Severity:   finding.SeverityError,
			Message:    "hardcoded AWS access key ID",
			Suggestion: "use IAM roles or environment credentials; never commit access keys",
			RetryHint:  "Remove the hardcoded AWS access key and use environment/role-based credentials.",
		},
		{
			// Failure mode 4: assigned secret literals (conservative shape).
			ID:         "no-hardcoded-secrets/assigned-literal",
			Pattern:    regexp.MustCompile(`(?i)(api[_-]?key|secret|token|passwd|password)\s*[:=]\s*["'][A-Za-z0-9_\-./+]{16,}["']`),
			Severity:   finding.SeverityError,
			Message:    "hardcoded secret literal",
			Suggestion: "move the value to configuration/secret storage and reference it indirectly",
			RetryHint:  "Replace the hardcoded secret literal with a reference to configuration or a secret manager.",
		},
		// The two rules below are the diff-local v1 heuristics (PRD blocker B4):
		// they ship before the tree-sitter spike and are therefore warn-severity
		// and deliberately conservative. The AST-precise versions (which know the
		// surrounding package's query style / real catch scopes) replace them
		// once spike-tree-sitter-ast lands.
		{
			// Failure mode 4: string-concatenated SQL.
			ID:         "no-sql-concat",
			Pattern:    regexp.MustCompile(`(?i)\b(select|insert\s+into|update|delete\s+from)\b.*?("\s*\+|\+\s*"|%s|%d|\$\{|f"|f')`),
			Severity:   finding.SeverityWarn,
			Message:    "SQL appears to be built by string concatenation/interpolation",
			Suggestion: "use parameterized queries instead of concatenating values into SQL",
			RetryHint:  "Rewrite this SQL to use bound parameters rather than string concatenation or interpolation.",
		},
		{
			// Failure mode 4: broad exception swallowing.
			ID:         "no-broad-catch",
			Pattern:    regexp.MustCompile(`(\bexcept\s*:|\bexcept\s+Exception\s*:\s*(pass)?\s*$|catch\s*\([^)]*\)\s*\{\s*\}|catch\s*\{\s*\})`),
			Severity:   finding.SeverityWarn,
			Message:    "broad exception catch that may swallow errors",
			Suggestion: "catch a specific exception and log or rethrow; don't silently swallow",
			RetryHint:  "Narrow this catch to the expected error type and log or rethrow instead of swallowing it.",
		},
	}
}
